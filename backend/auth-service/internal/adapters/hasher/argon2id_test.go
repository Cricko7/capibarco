package hasher_test

import (
	"strings"
	"testing"

	"github.com/hackathon/authsvc/internal/adapters/hasher"
)

func TestArgon2idHashesAndVerifiesPassword(t *testing.T) {
	h := hasher.NewArgon2id(hasher.Argon2idParams{
		MemoryKiB:   64 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})

	encoded, err := h.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=65536,t=1,p=1$") {
		t.Fatalf("unexpected encoded hash format: %s", encoded)
	}

	ok, err := h.Verify("correct horse battery staple", encoded)
	if err != nil {
		t.Fatalf("verify password: %v", err)
	}
	if !ok {
		t.Fatal("expected password to verify")
	}

	ok, err = h.Verify("wrong password", encoded)
	if err != nil {
		t.Fatalf("verify wrong password: %v", err)
	}
	if ok {
		t.Fatal("expected wrong password to fail verification")
	}
}
