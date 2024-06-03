package single

import (
	"context"
	"sync"

	"github.com/ethpandaops/tracoor/pkg/agent"
	"github.com/ethpandaops/tracoor/pkg/server"
	"github.com/sirupsen/logrus"
)

type Single struct {
	log    logrus.FieldLogger
	config *Config
}

func New(log logrus.FieldLogger, config *Config) *Single {
	return &Single{
		log:    log,
		config: config,
	}
}

func (s *Single) Start(ctx context.Context) error {
	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sserver, err := server.NewServer(ctx, s.log.WithField("container", "server"), s.config.Server)
	if err != nil {
		s.log.Fatal(err)
	}

	var wg sync.WaitGroup

	// Start server
	wg.Add(1)

	go func() {
		defer wg.Done()
		s.log.Info("Starting server")

		if err := sserver.Start(ctx); err != nil {
			s.log.Fatal(err)
		}

		s.log.Info("tracoor server exited.")

		// Cancel the context to signal all agents to exit
		cancel()
	}()

	// Wait for the server to start before starting agents
	go func() {
		<-sserver.Started

		// Start all the agents
		for _, cfg := range s.config.Agents {
			wg.Add(1)

			go func(cfg *agent.Config) {
				defer wg.Done()

				a, err := agent.New(ctx, s.log.WithField("container", "agent").WithField("name", cfg.Name), cfg)
				if err != nil {
					s.log.Fatal(err)
				}

				if err := a.Start(ctx); err != nil {
					s.log.Fatal(err)
				}

				s.log.Info("tracoor agent exited!")
			}(cfg)
		}
	}()

	wg.Wait()

	s.log.Info("tracoor single exited!")

	return nil
}
