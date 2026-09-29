package services

import "context"

type Name string

// FailureHandler is told that a service has given up. A service that can never
// become ready describes one node, so the failure travels to whoever owns that
// node rather than ending the process everything else is running in.
type FailureHandler func(ctx context.Context, err error)

type Service interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Ready(ctx context.Context) error
	OnReady(ctx context.Context, cb func(ctx context.Context) error)
	// OnFailure registers a handler for a service that has stopped trying. A
	// service that reports a failure will never report ready.
	OnFailure(cb FailureHandler)
	Name() Name
}
