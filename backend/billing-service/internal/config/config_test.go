package config_test

import (
	"testing"

	"github.com/petmatch/petmatch/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigFile(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load("../../configs/config.yaml")

	require.NoError(t, err)
	require.Equal(t, "billing-service", cfg.App.Name)
	require.Equal(t, ":9090", cfg.GRPC.Addr)
	require.Equal(t, "mock", cfg.Payment.Provider)
	require.Positive(t, cfg.GRPC.ShutdownTimeout)
}
