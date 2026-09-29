package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// Start feeds its errgroup result through isCleanShutdown, so anything treated
// as a failure here becomes a non-zero exit on an ordinary SIGTERM.
func TestIsCleanShutdown(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		clean bool
	}{
		{name: "no error", err: nil, clean: true},
		{name: "http listener closed", err: http.ErrServerClosed, clean: true},
		{name: "wrapped http listener closed", err: fmt.Errorf("gateway: %w", http.ErrServerClosed), clean: true},
		{name: "grpc server stopped", err: grpc.ErrServerStopped, clean: true},
		{name: "context canceled", err: context.Canceled, clean: true},
		{name: "wrapped context canceled", err: fmt.Errorf("indexer: %w", context.Canceled), clean: true},
		{name: "listen failure", err: errors.New("failed to listen: address already in use"), clean: false},
		{name: "deadline exceeded", err: context.DeadlineExceeded, clean: false},
		{name: "shutdown failure", err: fmt.Errorf("failed to stop indexer: %w", errors.New("database is locked")), clean: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.clean, isCleanShutdown(test.err))
		})
	}
}
