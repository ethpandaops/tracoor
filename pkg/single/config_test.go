package single

import (
	"testing"

	"github.com/ethpandaops/tracoor/pkg/agent"
	"github.com/ethpandaops/tracoor/pkg/agent/indexer"
	"github.com/ethpandaops/tracoor/pkg/server"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/stretchr/testify/require"
)

func TestApplySharedShutdownTimeout(t *testing.T) {
	config := &Config{
		Shared: &SharedConfig{
			LoggingLevel:           "debug",
			MetricsAddr:            ":9091",
			Indexer:                &indexer.Config{},
			Store:                  &store.Config{},
			ShutdownTimeoutSeconds: 30,
		},
		Server: &server.Config{},
		Agents: []*agent.Config{
			{Name: "inherits"},
			{Name: "overrides", ShutdownTimeoutSeconds: 5},
		},
	}

	require.NoError(t, config.ApplyShared())

	require.Equal(t, 30, config.Agents[0].ShutdownTimeoutSeconds)
	require.Equal(t, 5, config.Agents[1].ShutdownTimeoutSeconds)
}
