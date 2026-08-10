package single

import (
	"context"
	"fmt"
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
		return fmt.Errorf("failed to create server: %w", err)
	}

	var wg sync.WaitGroup

	// Written before wg.Done and read after wg.Wait, so no further synchronisation is needed.
	var serverErr error

	// Start server
	wg.Add(1)

	go func() {
		defer wg.Done()

		s.log.Info("Starting server")

		if err := sserver.Start(ctx); err != nil {
			serverErr = fmt.Errorf("server exited with an error: %w", err)

			s.log.WithError(err).Error("Server exited with an error")
		}

		s.log.Info("tracoor server exited.")

		// Cancel the context to signal all agents to exit
		cancel()
	}()

	// Wait for the server to start before starting agents
	go func() {
		select {
		case <-ctx.Done():
			// Shut down before the server ever came up, so there is nothing to
			// start the agents against.
			return
		case <-sserver.Started:
		}

		// Start all the agents
		for _, cfg := range s.config.Agents {
			wg.Add(1)

			go func(cfg *agent.Config) {
				defer wg.Done()

				log := s.log.WithField("container", "agent").WithField("name", cfg.Name)

				// One agent failing must not take the server or the other agents with it.
				a, err := agent.New(ctx, log, cfg)
				if err != nil {
					log.WithError(err).Error("Failed to create agent")

					return
				}

				if err := a.Start(ctx); err != nil {
					log.WithError(err).Error("Agent exited with an error")

					return
				}

				log.Info("tracoor agent exited!")
			}(cfg)
		}
	}()

	wg.Wait()

	s.log.Info("tracoor single exited!")

	return serverErr
}
