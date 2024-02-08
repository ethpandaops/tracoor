package server

import (
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/server/service"
	"github.com/ethpandaops/tracoor/pkg/store"
)

type Config struct {
	// The address to listen for GRPC requests on.
	Addr string `yaml:"addr" default:":8080"`
	// PreStopSleepSeconds is the number of seconds to sleep before stopping.
	// Useful for giving kubernetes time to drain connections.
	// This sleep will happen after a SIGTERM is received, and will
	// delay the shutdown of the server and all of it's components.
	// Note: Do not set this to a value greater than the kubernetes
	// terminationGracePeriodSeconds.
	PreStopSleepSeconds int `yaml:"preStopSleepSeconds" default:"0"`
	// GatewayAddr is the address to listen on for the grpc-gateway.
	GatewayAddr string `yaml:"gatewayAddr" default:":7007"`
	// MetricsAddr is the address to listen on for metrics.
	MetricsAddr string `yaml:"metricsAddr" default:":9090"`
	// PProfAddr is the address to listen on for pprof.
	PProfAddr *string `yaml:"pprofAddr"`
	// LoggingLevel is the logging level to use.
	LoggingLevel string `yaml:"logging" default:"info"`
	// Store is the cache configuration.
	Persistence persistence.Config `yaml:"persistence"`
	// Store is the cache configuration.
	Store store.Config `yaml:"store"`
	// Services is the list of services to run.
	Services service.Config `yaml:"services"`
}

func (c *Config) Validate() error {
	if err := c.Services.Validate(); err != nil {
		return err
	}

	if err := c.Persistence.Validate(); err != nil {
		return err
	}

	if err := c.Store.Validate(); err != nil {
		return err
	}

	return nil
}
