package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// nodeStopTimeout bounds tearing down the ethereum node transports. It runs
// after the workers have unwound, on a context of its own, because the agent's
// context is already cancelled by the time we get here.
const nodeStopTimeout = 5 * time.Second

// pprofShutdownTimeout bounds closing the debug server. It holds nothing the
// agent cares about, so it is never worth waiting long for.
const pprofShutdownTimeout = 5 * time.Second

// workerGroup tracks the background loops an agent owns so a shutdown can wait
// for them instead of abandoning them mid-fetch. Once the wait has begun it
// refuses new workers: starting work nothing will wait for is pointless, and
// growing a WaitGroup from zero while it is being waited on is a race.
type workerGroup struct {
	mu       sync.Mutex
	draining bool
	wg       sync.WaitGroup
}

// Go runs fn as a tracked worker, unless the agent is already shutting down.
func (g *workerGroup) Go(fn func()) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.draining {
		return
	}

	g.wg.Add(1)

	go func() {
		defer g.wg.Done()

		fn()
	}()
}

// Wait stops accepting workers and waits up to timeout for the running ones to
// finish. It reports whether they all did; false means the remainder was
// abandoned, which is safe because artifacts are written to the store before
// they are indexed, so the worst a kill leaves behind is an unreferenced object.
func (g *workerGroup) Wait(timeout time.Duration) bool {
	g.mu.Lock()
	g.draining = true
	g.mu.Unlock()

	done := make(chan struct{})

	go func() {
		defer close(done)

		g.wg.Wait()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}

// shutdown unwinds the agent in the only safe order. The context is already
// cancelled by the time this runs, which aborts in-flight requests; the queue
// workers then get a bounded grace period to finish what they can; and the
// ethereum nodes are stopped last, because stopping them tears down the
// transports those workers are still reading through.
func (s *agent) shutdown(ctx context.Context) {
	s.scheduler.Stop()

	timeout := s.Config.ShutdownTimeout()

	if s.workers.Wait(timeout) {
		s.log.Info("All queue workers finished")
	} else {
		s.log.
			WithField("timeout", timeout.String()).
			Warn("Queue workers did not finish within the shutdown grace period, abandoning them")
	}

	// ctx is what the workers just unwound on, so stopping the nodes needs a
	// context that outlives it.
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), nodeStopTimeout)
	defer cancel()

	if err := s.stopNodes(stopCtx); err != nil {
		s.log.WithError(err).Warn("Failed to cleanly stop the ethereum nodes")
	}
}

func (s *agent) stopNodes(ctx context.Context) error {
	if s.node == nil {
		return nil
	}

	var errs []error

	if beaconNode := s.node.Beacon(); beaconNode != nil {
		if err := beaconNode.Node().Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("beacon node: %w", err))
		}
	}

	if executionNode := s.node.Execution(); executionNode != nil {
		if err := executionNode.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("execution node: %w", err))
		}
	}

	return errors.Join(errs...)
}
