// Package problem writes RFC 7807 problem responses.
package problem

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/petmatch/petmatch/internal/app/gateway"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Details is an RFC 7807 problem response.
type Details struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// Abort maps err to a problem response and aborts the Gin context.
func Abort(c *gin.Context, err error) {
	statusCode, title, typ := mapError(err)
	c.AbortWithStatusJSON(statusCode, Details{
		Type:     typ,
		Title:    title,
		Status:   statusCode,
		Detail:   err.Error(),
		Instance: c.Request.URL.Path,
	})
	c.Header("Content-Type", "application/problem+json")
}

func mapError(err error) (int, string, string) {
	switch {
	case errors.Is(err, gateway.ErrUnauthenticated):
		return http.StatusUnauthorized, "Unauthorized", "https://api.petmatch.local/errors/unauthorized"
	case errors.Is(err, gateway.ErrPermissionDenied):
		return http.StatusForbidden, "Forbidden", "https://api.petmatch.local/errors/forbidden"
	case errors.Is(err, gateway.ErrInvalidInput), errors.Is(err, gateway.ErrIdempotencyKeyRequired):
		return http.StatusBadRequest, "Bad Request", "https://api.petmatch.local/errors/bad-request"
	case errors.Is(err, gateway.ErrDependencyDisabled):
		return http.StatusNotImplemented, "Not Implemented", "https://api.petmatch.local/errors/dependency-disabled"
	case errors.Is(err, gateway.ErrRateLimited):
		return http.StatusTooManyRequests, "Too Many Requests", "https://api.petmatch.local/errors/rate-limit"
	}
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unauthenticated:
			return http.StatusUnauthorized, "Unauthorized", "https://api.petmatch.local/errors/unauthorized"
		case codes.PermissionDenied:
			return http.StatusForbidden, "Forbidden", "https://api.petmatch.local/errors/forbidden"
		case codes.NotFound:
			return http.StatusNotFound, "Not Found", "https://api.petmatch.local/errors/not-found"
		case codes.InvalidArgument, codes.FailedPrecondition, codes.AlreadyExists:
			return http.StatusBadRequest, "Bad Request", "https://api.petmatch.local/errors/bad-request"
		case codes.ResourceExhausted:
			return http.StatusTooManyRequests, "Too Many Requests", "https://api.petmatch.local/errors/rate-limit"
		case codes.Unavailable, codes.DeadlineExceeded:
			return http.StatusBadGateway, "Bad Gateway", "https://api.petmatch.local/errors/downstream-unavailable"
		}
	}
	return http.StatusInternalServerError, "Internal Server Error", "https://api.petmatch.local/errors/internal"
}
