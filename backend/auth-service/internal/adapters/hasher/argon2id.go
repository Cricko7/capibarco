package hasher

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2idParams controls password hashing cost.
type Argon2idParams struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultArgon2idParams is intentionally strong for production servers.
func DefaultArgon2idParams() Argon2idParams {
	return Argon2idParams{
		MemoryKiB:   128 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// Argon2id hashes passwords using PHC string format.
type Argon2id struct {
	params Argon2idParams
}

// NewArgon2id creates a password hasher.
func NewArgon2id(params Argon2idParams) *Argon2id {
	return &Argon2id{params: params}
}

// Hash hashes a password.
func (h *Argon2id) Hash(password string) (string, error) {
	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, h.params.Iterations, h.params.MemoryKiB, h.params.Parallelism, h.params.KeyLength)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		h.params.MemoryKiB,
		h.params.Iterations,
		h.params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify compares a password with an encoded Argon2id hash.
func (h *Argon2id) Verify(password string, encodedHash string) (bool, error) {
	params, salt, key, err := parsePHC(encodedHash)
	if err != nil {
		return false, err
	}
	candidate := argon2.IDKey([]byte(password), salt, params.Iterations, params.MemoryKiB, params.Parallelism, params.KeyLength)
	return subtle.ConstantTimeCompare(candidate, key) == 1, nil
}

func parsePHC(encodedHash string) (Argon2idParams, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return Argon2idParams{}, nil, nil, errors.New("invalid argon2id hash")
	}
	settings := strings.Split(parts[3], ",")
	if len(settings) != 3 {
		return Argon2idParams{}, nil, nil, errors.New("invalid argon2id params")
	}
	memory, err := parseSetting(settings[0], "m")
	if err != nil {
		return Argon2idParams{}, nil, nil, err
	}
	iterations, err := parseSetting(settings[1], "t")
	if err != nil {
		return Argon2idParams{}, nil, nil, err
	}
	parallelism, err := parseSetting(settings[2], "p")
	if err != nil {
		return Argon2idParams{}, nil, nil, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Argon2idParams{}, nil, nil, fmt.Errorf("decode salt: %w", err)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Argon2idParams{}, nil, nil, fmt.Errorf("decode key: %w", err)
	}
	return Argon2idParams{
		MemoryKiB:   uint32(memory),
		Iterations:  uint32(iterations),
		Parallelism: uint8(parallelism),
		SaltLength:  uint32(len(salt)),
		KeyLength:   uint32(len(key)),
	}, salt, key, nil
}

func parseSetting(raw string, key string) (uint64, error) {
	prefix := key + "="
	if !strings.HasPrefix(raw, prefix) {
		return 0, fmt.Errorf("missing argon2id setting %s", key)
	}
	value, err := strconv.ParseUint(strings.TrimPrefix(raw, prefix), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse argon2id setting %s: %w", key, err)
	}
	return value, nil
}
