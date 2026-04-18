package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load("")
	require.NoError(t, err)
	require.Equal(t, "analytics-service", cfg.App.Name)
	require.Equal(t, ":19096", cfg.GRPC.Addr)
	require.NotEmpty(t, cfg.Kafka.Brokers)
}
