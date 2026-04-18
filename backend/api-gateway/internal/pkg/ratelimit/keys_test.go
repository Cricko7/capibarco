package ratelimit

import (
	"testing"

	"github.com/petmatch/petmatch/internal/app/gateway"
	"github.com/stretchr/testify/require"
)

func TestKeysForRequestIncludesIPActorAndRoles(t *testing.T) {
	keys := KeysForRequest("10.0.0.1", gateway.Principal{
		ActorID: "user-1",
		Roles:   []string{"admin", "shelter"},
	})

	require.Equal(t, []string{
		"ip:10.0.0.1",
		"actor:user-1",
		"role:admin",
		"role:shelter",
	}, keys)
}

func TestKeysForRequestOmitsEmptyValues(t *testing.T) {
	keys := KeysForRequest("", gateway.Principal{})
	require.Empty(t, keys)
}
