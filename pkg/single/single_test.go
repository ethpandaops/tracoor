package single

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/ethpandaops/tracoor/pkg/agent"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func testSingle(agents ...string) *Single {
	log := logrus.New()
	log.SetOutput(io.Discard)

	cfgs := make([]*agent.Config, 0, len(agents))

	for _, name := range agents {
		cfgs = append(cfgs, &agent.Config{Name: name})
	}

	return &Single{
		log:    log,
		config: &Config{Agents: cfgs},
	}
}

func TestIsCleanShutdown(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		clean bool
	}{
		{name: "no error", err: nil, clean: true},
		{name: "server closed", err: http.ErrServerClosed, clean: true},
		{name: "wrapped server closed", err: fmt.Errorf("gateway: %w", http.ErrServerClosed), clean: true},
		{name: "context canceled", err: context.Canceled, clean: true},
		{name: "real error", err: errors.New("failed to listen: address already in use"), clean: false},
		{name: "deadline exceeded", err: context.DeadlineExceeded, clean: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.clean, isCleanShutdown(test.err))
		})
	}
}

// A clean shutdown must not be reported as a failure - cmd/single.go turns a
// non-nil error into a fatal, and the process would exit 1 on every restart.
func TestRunReturnsNilOnCleanShutdown(t *testing.T) {
	s := testSingle("a")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopServer := make(chan struct{})
	started := make(chan struct{})

	srv := supervisedServer{
		run: func() error {
			<-stopServer

			return http.ErrServerClosed
		},
		stop: sync.OnceFunc(func() { close(stopServer) }),
	}

	agentStarted := make(chan struct{})

	done := make(chan error, 1)

	go func() {
		done <- s.run(ctx, srv, started, func(agentCtx context.Context, _ *agent.Config) {
			close(agentStarted)

			<-agentCtx.Done()
		})
	}()

	close(started)

	requireClosed(t, agentStarted, "agent did not start")

	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("run did not return after the context was cancelled")
	}
}

// Agents index what they have already uploaded, so the server has to outlive
// their drain. Anything else leaves objects in the store with no row.
func TestRunStopsServerAfterAgentsHaveDrained(t *testing.T) {
	s := testSingle("a", "b")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		mu    sync.Mutex
		exits []string
	)

	recordExit := func(name string) {
		mu.Lock()
		defer mu.Unlock()

		exits = append(exits, name)
	}

	stopServer := make(chan struct{})
	started := make(chan struct{})

	srv := supervisedServer{
		run: func() error {
			<-stopServer

			recordExit("server")

			return nil
		},
		stop: sync.OnceFunc(func() { close(stopServer) }),
	}

	var running sync.WaitGroup

	running.Add(len(s.config.Agents))

	startAgent := func(agentCtx context.Context, cfg *agent.Config) {
		running.Done()

		<-agentCtx.Done()

		// Stand in for the drain: in-flight uploads finishing and being indexed
		// after the signal but before the agent gives up.
		time.Sleep(100 * time.Millisecond)

		recordExit(cfg.Name)
	}

	done := make(chan error, 1)

	go func() {
		done <- s.run(ctx, srv, started, startAgent)
	}()

	close(started)

	running.Wait()

	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("run did not return after the context was cancelled")
	}

	require.Len(t, exits, 3)
	require.Equal(t, "server", exits[len(exits)-1], "server stopped before the agents finished draining")
}

// A server that fails still has to bring the process down, drain ordering or
// not, and the failure has to reach the caller.
func TestRunReturnsServerFailureWithoutHanging(t *testing.T) {
	s := testSingle("a")

	failure := errors.New("failed to listen")

	srv := supervisedServer{
		run:  func() error { return failure },
		stop: func() {},
	}

	done := make(chan error, 1)

	go func() {
		done <- s.run(context.Background(), srv, make(chan struct{}), func(agentCtx context.Context, _ *agent.Config) {
			<-agentCtx.Done()
		})
	}()

	select {
	case err := <-done:
		require.ErrorIs(t, err, failure)
	case <-time.After(5 * time.Second):
		t.Fatal("run did not return after the server failed")
	}
}

func requireClosed(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal(msg)
	}
}
