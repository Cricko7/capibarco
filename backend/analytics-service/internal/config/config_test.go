package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load("")
	require.NoError(t, err)
	require.Equal(t, "analytics-service", cfg.App.Name)
	require.NotEmpty(t, cfg.Postgres.DSN)
}
