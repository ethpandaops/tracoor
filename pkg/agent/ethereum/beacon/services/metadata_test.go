package services

import (
	"context"
	goerrors "errors"
	"io"
	"testing"
	"time"

	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// stubBeaconNode answers only the health question. Anything else the metadata
// service might reach for is deliberately absent: a test that needs it is
// testing something this stub has no opinion about.
type stubBeaconNode struct {
	beacon.Node

	healthy bool
}

func (n stubBeaconNode) Healthy() bool { return n.healthy }

func newTestMetadataService(healthy bool) MetadataService {
	log := logrus.New()
	log.SetOutput(io.Discard)

	return NewMetadataService(log, stubBeaconNode{healthy: healthy}, "")
}

func TestWaitForHealthyBeaconNodeReturnsAnErrorInsteadOfEndingTheProcess(t *testing.T) {
	service := newTestMetadataService(false)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// A node that never becomes healthy is one node's problem. Reaching this
	// assertion at all is the point: the old behaviour exited the process, and
	// in single mode that took the server and every other agent with it.
	err := service.WaitForHealthyBeaconNode(ctx)
	require.Error(t, err)
	require.ErrorContains(t, err, "beacon node did not become healthy")
}

func TestWaitForHealthyBeaconNodeReturnsOnceTheNodeIsHealthy(t *testing.T) {
	service := newTestMetadataService(true)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	require.NoError(t, service.WaitForHealthyBeaconNode(ctx))
}

func TestMetadataFailureReachesTheHandlersThatOwnTheNode(t *testing.T) {
	service := newTestMetadataService(false)

	failure := goerrors.New("gave up on the node")

	var got []error

	service.OnFailure(func(_ context.Context, err error) {
		got = append(got, err)
	})

	service.OnFailure(func(_ context.Context, err error) {
		got = append(got, err)
	})

	service.reportFailure(context.Background(), failure)

	require.Len(t, got, 2, "every owner registered for the failure is told about it")

	for _, err := range got {
		require.ErrorIs(t, err, failure)
	}
}

func TestMetadataServiceIsNotReadyWithoutItsMetadata(t *testing.T) {
	service := newTestMetadataService(true)

	// Ready is what the retry loop keeps asking, and what has to keep failing
	// rather than ending the process while a node is unusable.
	require.Error(t, service.Ready(context.Background()))
}
