package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (s *agent) fetchAndIndexBeaconState(ctx context.Context, slot phase0.Slot) error {
	// Fetch the state root
	root, err := s.node.Beacon().Node().FetchBeaconStateRoot(ctx, fmt.Sprintf("%d", slot))
	if err != nil {
		return err
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
		Slot:      uint64(slot),
		StateRoot: rootAsString,
	})
	if err != nil {
		s.log.WithField("state_root", rootAsString).WithField("slot", slot).WithError(err).Error("Failed to check if beacon state is already indexed")
	}

	if rsp != nil && len(rsp.BeaconStates) > 0 {
		s.log.WithField("state_root", rootAsString).WithField("slot", slot).Debug("Beacon state already indexed")

		return nil
	}

	now := time.Now()

	// Fetch the state
	state, err := s.node.Beacon().Node().FetchRawBeaconState(ctx, rootAsString, "application/octet-stream")
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

	s.log.WithField("state_root", root).WithField("slot", slot).Debug("Indexed beacon state")

	return nil
}
