package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/beacon/services"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (s *agent) fetchAndIndexBeaconState(ctx context.Context, slot phase0.Slot) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	root, err := s.node.Beacon().Node().FetchBeaconStateRoot(ctx, fmt.Sprintf("%d", slot))
	if err != nil {
		return errors.Wrap(err, "failed to fetch beacon state root")
	}

	rootAsString := fmt.Sprintf("%#x", root)

	location := CreateBeaconStateFileName(
		s.Config.Name,
		string(s.node.Beacon().Metadata().Network.Name),
		slot,
		rootAsString,
	)

	location = fmt.Sprintf("%s.ssz", location)

	// Check if we've somehow already indexed this beacon state
	rsp, err := s.indexer.ListBeaconState(ctx, &indexer.ListBeaconStateRequest{
		Node:      s.Config.Name,
		StateRoot: rootAsString,
		Slot:      uint64(slot),
	})
	if err != nil {
		s.log.
			WithField("state_root", rootAsString).
			WithField("slot", slot).
			WithError(err).
			Error("Failed to check if beacon state is already indexed")
	}

	if rsp != nil && len(rsp.BeaconStates) > 0 {
		s.log.
			WithField("state_root", rootAsString).
			WithField("slot", slot).
			Debug("Beacon state already indexed")

		return nil
	}

	now := time.Now()

	stateID := rootAsString

	client := string(s.node.Beacon().Metadata().Client(ctx))

	if client == string(services.ClientLodestar) ||
		client == string(services.ClientPrysm) {
		// Lodestar/prysm requires us to fetch the state id by slot
		stateID = fmt.Sprintf("%d", slot)
	}

	// Fetch the state
	state, err := s.node.Beacon().Node().FetchRawBeaconState(ctx, stateID, "application/octet-stream")
	if err != nil {
		return err
	}

	s.log.WithField("location", location).Debug("Saving beacon state")

	// Upload the state to the store
	location, err = s.store.SaveBeaconState(ctx, &state, location)
	if err != nil {
		return err
	}

	// Sleep for 1s to give the store time to update
	time.Sleep(1 * time.Second)

	spec, err := s.node.Beacon().Node().Spec()
	if err != nil {
		return err
	}

	req := &indexer.CreateBeaconStateRequest{
		Node:        wrapperspb.String(s.Config.Name),
		Network:     wrapperspb.String(string(s.node.Beacon().Metadata().Network.Name)),
		Slot:        wrapperspb.UInt64(uint64(slot)),
		Epoch:       wrapperspb.UInt64(uint64(slot) / uint64(spec.SlotsPerEpoch)),
		StateRoot:   wrapperspb.String(rootAsString),
		Location:    wrapperspb.String(location),
		NodeVersion: wrapperspb.String(s.node.Beacon().Metadata().NodeVersion(ctx)),
		BeaconImplementation: wrapperspb.String(
			s.node.Beacon().Metadata().Client(ctx),
		),
		FetchedAt: timestamppb.New(now),
	}

	// Index the state
	if _, err := s.indexer.CreateBeaconState(ctx, req); err != nil {
		return err
	}

	s.log.
		WithField("state_root", rootAsString).
		WithField("slot", slot).
		Debug("Indexed beacon state")

	return nil
}
