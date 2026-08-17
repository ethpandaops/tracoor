package single

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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

// supervisedServer is the slice of the server's lifecycle that single
// orchestrates: run blocks until the server exits, stop asks it to.
type supervisedServer struct {
	run  func() error
	stop context.CancelFunc
}

func (s *Single) Start(ctx context.Context) error {
	// The server's lifetime is deliberately detached from ctx. Agents write an
	// artifact to the store first and index it afterwards, so an indexer that
	// disappears the instant a signal arrives leaves everything the agents
	// upload during their grace period in the store with no row referencing it.
	// run stops the server explicitly, once the agents have drained.
	serverCtx, stopServer := context.WithCancel(context.WithoutCancel(ctx))
	defer stopServer()

	sserver, err := server.NewServer(serverCtx, s.log.WithField("container", "server"), s.config.Server)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	return s.run(ctx, supervisedServer{
		run:  func() error { return sserver.Start(serverCtx) },
		stop: stopServer,
	}, sserver.Started, s.startAgent)
}

// run supervises the server and the agents. The shutdown order is the point of
// it: a signal unwinds the agents only, and the server is stopped once they
// have all exited, so an upload that finishes inside an agent's grace period
// can still be indexed. A server that exits on its own - cleanly or not - takes
// the agents with it, so a failed server can never strand the process.
func (s *Single) run(ctx context.Context, srv supervisedServer, started <-chan struct{}, startAgent func(context.Context, *agent.Config)) error {
	agentCtx, stopAgents := context.WithCancel(ctx)
	defer stopAgents()

	var (
		serverWG sync.WaitGroup
		agentsWG sync.WaitGroup
	)

	// Written before serverWG.Done and read after serverWG.Wait, so no further
	// synchronisation is needed.
	var serverErr error

	serverWG.Add(1)

	go func() {
		defer serverWG.Done()

		s.log.Info("Starting server")

		if err := srv.run(); !isCleanShutdown(err) {
			serverErr = fmt.Errorf("server exited with an error: %w", err)

			s.log.WithError(err).Error("Server exited with an error")
		}

		s.log.Info("tracoor server exited.")

		// Agents have nothing to index against without the server, so its exit is
		// their cue to stop too.
		stopAgents()
	}()

	agentsWG.Add(1)

	go func() {
		defer agentsWG.Done()

		select {
		case <-agentCtx.Done():
			// Shut down before the server ever came up, so there is nothing to
			// start the agents against.
			return
		case <-started:
		}

		// Started while this goroutine still holds a count of its own, so the
		// wait below can never observe an empty group early.
		for _, cfg := range s.config.Agents {
			agentsWG.Add(1)

			go func() {
				defer agentsWG.Done()

				startAgent(agentCtx, cfg)
			}()
		}
	}()

	// Agents first: they are the ones holding uploads that still need indexing.
	// Their drain is bounded by their own shutdown grace period.
	agentsWG.Wait()

	s.log.Info("All agents exited, stopping the server")

	srv.stop()

	serverWG.Wait()

	s.log.Info("tracoor single exited!")

	return serverErr
}

func (s *Single) startAgent(ctx context.Context, cfg *agent.Config) {
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
}

// isCleanShutdown reports whether err is what a shutdown we asked for looks
// like, rather than a failure the process should exit non-zero on. The server
// filters these at its own boundary; this is the backstop that keeps a normal
// stop from reaching cmd as a fatal.
func isCleanShutdown(err error) bool {
	return err == nil ||
		errors.Is(err, http.ErrServerClosed) ||
		errors.Is(err, context.Canceled)
}
