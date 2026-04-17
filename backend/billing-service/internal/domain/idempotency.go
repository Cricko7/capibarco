package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

func HashIdempotencyKey(scope string, key string) (string, error) {
	scope = strings.TrimSpace(scope)
	key = strings.TrimSpace(key)
	if scope == "" || key == "" {
		return "", fmt.Errorf("%w: idempotency scope and key are required", ErrValidation)
	}
	sum := sha256.Sum256([]byte(scope + "\x00" + key))
	return hex.EncodeToString(sum[:]), nil
}
