package httpserver

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/petmatch/petmatch/internal/app/gateway"
	kafkaevents "github.com/petmatch/petmatch/internal/infra/kafka"
	"github.com/petmatch/petmatch/internal/pkg/problem"
	"github.com/petmatch/petmatch/internal/pkg/ratelimit"
	"github.com/petmatch/petmatch/internal/pkg/requestid"
)

func (s *Server) recoverer() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		s.logger.ErrorContext(c.Request.Context(), "panic recovered in http handler", "panic", recovered)
		problem.Abort(c, fmt.Errorf("internal server error"))
	})
}

func (s *Server) requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Header("X-Request-ID", rid)
		c.Request = c.Request.WithContext(requestid.With(c.Request.Context(), rid))
		c.Next()
	}
}

func (s *Server) securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		c.Next()
	}
}

func (s *Server) cors() gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(s.cfg.HTTP.CORSOrigins))
	for _, origin := range s.cfg.HTTP.CORSOrigins {
		allowed[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok || len(allowed) == 0 {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Idempotency-Key, X-Request-ID, X-Guest-Session-Token")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (s *Server) sizeLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, s.cfg.HTTP.MaxBodyBytes)
		c.Next()
	}
}

func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		s.metrics.ObserveHTTP(c.Request.Method, route, c.Writer.Status(), time.Since(started))
		s.logger.InfoContext(c.Request.Context(), "http request completed",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"route", route,
			"status", c.Writer.Status(),
			"request_id", requestid.From(c.Request.Context()),
			"duration_ms", time.Since(started).Milliseconds(),
		)
	}
}

func (s *Server) auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		bearer := bearerToken(c)
		var principal gateway.Principal
		var err error
		if bearer != "" {
			principal, err = s.app.ValidateBearer(c.Request.Context(), bearer)
		} else if guest := c.GetHeader("X-Guest-Session-Token"); guest != "" {
			principal, err = s.app.ValidateGuest(guest)
		} else {
			err = gateway.ErrUnauthenticated
		}
		if err != nil {
			s.publishRejected(c, http.StatusUnauthorized, err.Error())
			problem.Abort(c, err)
			return
		}
		c.Request = c.Request.WithContext(gateway.WithPrincipal(c.Request.Context(), principal))
		c.Next()
	}
}

func (s *Server) rateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, _ := gateway.PrincipalFromContext(c.Request.Context())
		for _, key := range ratelimit.KeysForRequest(c.ClientIP(), principal) {
			limit := s.limitFor(key, principal)
			decision, err := s.limiter.Allow(c.Request.Context(), key, limit, s.cfg.Rate.Window)
			if err != nil {
				problem.Abort(c, fmt.Errorf("rate limiter unavailable: %w", err))
				return
			}
			if !decision.Allowed {
				s.metrics.RateLimitHits.WithLabelValues(bucketKind(key)).Inc()
				c.Header("Retry-After", strconvRetryAfter(decision.RetryAfter))
				s.publishRejected(c, http.StatusTooManyRequests, "rate limit exceeded")
				problem.Abort(c, gateway.ErrRateLimited)
				return
			}
		}
		c.Next()
	}
}

func (s *Server) limitFor(key string, principal gateway.Principal) int64 {
	if strings.HasPrefix(key, "ip:") {
		return s.cfg.Rate.IPLimit
	}
	if strings.HasPrefix(key, "actor:") {
		if principal.IsGuest {
			return s.cfg.Rate.GuestLimit
		}
		return s.cfg.Rate.ActorLimit
	}
	if key == "role:admin" {
		return s.cfg.Rate.AdminRoleLimit
	}
	return s.cfg.Rate.RoleLimit
}

func bucketKind(key string) string {
	switch {
	case strings.HasPrefix(key, "ip:"):
		return "ip"
	case strings.HasPrefix(key, "actor:"):
		return "actor"
	case strings.HasPrefix(key, "role:"):
		return "role"
	default:
		return "unknown"
	}
}

func bearerToken(c *gin.Context) string {
	if value := strings.TrimSpace(c.GetHeader("Authorization")); value != "" {
		if strings.HasPrefix(strings.ToLower(value), "bearer ") {
			return strings.TrimSpace(value[7:])
		}
		return ""
	}

	value := strings.TrimSpace(c.Query("access_token"))
	if value == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return value
}

func strconvRetryAfter(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return fmt.Sprintf("%d", seconds)
}

func (s *Server) publishRejected(c *gin.Context, status int, reason string) {
	principal, _ := gateway.PrincipalFromContext(c.Request.Context())
	payload := map[string]any{
		"request_id":  requestid.From(c.Request.Context()),
		"actor_id":    principal.ActorID,
		"route":       c.FullPath(),
		"method":      c.Request.Method,
		"status_code": status,
		"reason":      reason,
		"ip":          c.ClientIP(),
		"user_agent":  c.Request.UserAgent(),
	}
	if err := s.publisher.Publish(c.Request.Context(), kafkaevents.TopicRequestRejected, requestid.From(c.Request.Context()), payload); err != nil {
		s.logger.WarnContext(c.Request.Context(), "publish gateway rejection event", "error", err)
	}
}
