package domain

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

var guestScopes = []string{"feed:read", "animal:read", "swipe:create"}

// GuestSession represents a stateless anonymous browsing session.
type GuestSession struct {
	ID            string    `json:"sid"`
	ActorID       string    `json:"aid"`
	DeviceID      string    `json:"did"`
	Locale        string    `json:"loc"`
	AllowedScopes []string  `json:"scp"`
	ExpiresAt     time.Time `json:"exp"`
}

// GuestSessionCodec creates and validates signed opaque guest-session tokens.
type GuestSessionCodec struct {
	secret []byte
	ttl    time.Duration
}

// NewGuestSessionCodec creates a guest-session codec.
func NewGuestSessionCodec(secret []byte, ttl time.Duration) *GuestSessionCodec {
	return &GuestSessionCodec{secret: append([]byte(nil), secret...), ttl: ttl}
}

// Create returns a signed guest-session token and its decoded session.
func (c *GuestSessionCodec) Create(deviceID string, locale string, now time.Time) (string, GuestSession, error) {
	session := GuestSession{
		ID:            randomID("gst"),
		ActorID:       randomID("guest"),
		DeviceID:      deviceID,
		Locale:        locale,
		AllowedScopes: append([]string(nil), guestScopes...),
		ExpiresAt:     now.Add(c.ttl).UTC(),
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return "", GuestSession{}, fmt.Errorf("marshal guest session: %w", err)
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := c.sign(encodedPayload)
	return encodedPayload + "." + signature, session, nil
}

// Parse verifies a guest-session token and returns its payload.
func (c *GuestSessionCodec) Parse(token string, now time.Time) (GuestSession, error) {
	payloadPart, signaturePart, ok := strings.Cut(token, ".")
	if !ok || payloadPart == "" || signaturePart == "" {
		return GuestSession{}, ErrInvalidGuestSession
	}
	if !hmac.Equal([]byte(c.sign(payloadPart)), []byte(signaturePart)) {
		return GuestSession{}, ErrInvalidGuestSession
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return GuestSession{}, fmt.Errorf("%w: decode payload", ErrInvalidGuestSession)
	}
	var session GuestSession
	if err := json.Unmarshal(payload, &session); err != nil {
		return GuestSession{}, fmt.Errorf("%w: decode json", ErrInvalidGuestSession)
	}
	if !now.Before(session.ExpiresAt) {
		return GuestSession{}, ErrGuestSessionExpired
	}
	return session, nil
}

func (c *GuestSessionCodec) sign(payload string) string {
	mac := hmac.New(sha256.New, c.secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func randomID(prefix string) string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		now := time.Now().UTC().UnixNano()
		return fmt.Sprintf("%s-%x", prefix, now)
	}
	return fmt.Sprintf("%s-%s", prefix, base64.RawURLEncoding.EncodeToString(raw[:]))
}
