package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/beacon/services"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

	client := s.node.Beacon().Metadata().Client(ctx)

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

	s.metrics.IncrementItemExported(BeaconStateQueue)

	s.log.
		WithField("state_root", rootAsString).
		WithField("slot", slot).
		Debug("Indexed beacon state")

	return nil
}

func (s *agent) fetchAndIndexBeaconBlock(ctx context.Context, slot phase0.Slot) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	block, err := s.node.Beacon().Node().FetchBlock(ctx, fmt.Sprintf("%d", slot))
	if err != nil {
		return errors.Wrap(err, "failed to fetch beacon block")
	}

	if block == nil {
		s.log.
			WithField("slot", slot).
			Debug("no beacon block found for slot")

		return nil
	}

	blockRoot, err := block.Root()
	if err != nil {
		return errors.Wrap(err, "failed to fetch beacon block")
	}

	blockRootAsString := blockRoot.String()

	location := CreateBeaconStateFileName(
		s.Config.Name,
		string(s.node.Beacon().Metadata().Network.Name),
		slot,
		blockRootAsString,
	)

	location = fmt.Sprintf("%s.ssz", location)

	// Check if we've somehow already indexed this beacon state
	rsp, err := s.indexer.ListBeaconBlock(ctx, &indexer.ListBeaconBlockRequest{
		Node:      s.Config.Name,
		BlockRoot: blockRootAsString,
		Slot:      uint64(slot),
	})
	if err != nil {
		s.log.
			WithField("block_root", blockRootAsString).
			WithField("slot", slot).
			WithError(err).
			Error("Failed to check if beacon block is already indexed")
	}

	if rsp != nil && len(rsp.BeaconBlocks) > 0 {
		s.log.
			WithField("block_root", blockRootAsString).
			WithField("slot", slot).
			Debug("Beacon block already indexed")

		return nil
	}

	now := time.Now()

	stateID := fmt.Sprintf("%d", slot)

	// Fetch the block
	blockRaw, err := s.node.Beacon().Node().FetchRawBlock(ctx, stateID, "application/octet-stream")
	if err != nil {
		return err
	}

	s.log.WithField("location", location).Debug("Saving beacon block")

	// Upload the block to the store
	location, err = s.store.SaveBeaconBlock(ctx, &blockRaw, location)
	if err != nil {
		return err
	}

	// Sleep for 1s to give the store time to update
	time.Sleep(1 * time.Second)

	spec, err := s.node.Beacon().Node().Spec()
	if err != nil {
		return err
	}

	req := &indexer.CreateBeaconBlockRequest{
		Node:        wrapperspb.String(s.Config.Name),
		Network:     wrapperspb.String(string(s.node.Beacon().Metadata().Network.Name)),
		Slot:        wrapperspb.UInt64(uint64(slot)),
		Epoch:       wrapperspb.UInt64(uint64(slot) / uint64(spec.SlotsPerEpoch)),
		BlockRoot:   wrapperspb.String(blockRootAsString),
		Location:    wrapperspb.String(location),
		NodeVersion: wrapperspb.String(s.node.Beacon().Metadata().NodeVersion(ctx)),
		BeaconImplementation: wrapperspb.String(
			s.node.Beacon().Metadata().Client(ctx),
		),
		FetchedAt: timestamppb.New(now),
	}

	// Index the block
	if _, err := s.indexer.CreateBeaconBlock(ctx, req); err != nil {
		return err
	}

	s.metrics.IncrementItemExported(BeaconBlockQueue)

	s.log.
		WithField("block_root", blockRootAsString).
		WithField("slot", slot).
		Debug("Indexed beacon block")

	return nil
}

func getBadBlocksFilePattern(client string) (*string, error) {
	var pattern string

	switch client {
	case string(services.ClientLighthouse):
		pattern = `^(\d+)_([^.]+)\.ssz$`
	case string(services.ClientNimbus):
		pattern = `^block-(\d+)-([^.]+)\.ssz$`
	default:
		return nil, errors.New("client does not have bad blocks available")
	}

	return &pattern, nil
}

func (s *agent) fetchAndIndexBeaconBadBlocks(ctx context.Context, path string) error {
	client := s.node.Beacon().Metadata().Client(ctx)

	// Verify the path is a directory
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("path %s is not a directory", path)
	}

	pattern, err := getBadBlocksFilePattern(client)
	if err != nil {
		return err
	}

	matcher := regexp.MustCompile(*pattern)

	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, file := range files {
		matches := matcher.FindStringSubmatch(file.Name())
		if len(matches) == 3 {
			filePath := filepath.Join(path, file.Name())

			// Parse 'slot' and 'blockRoot' from the file name
			slotI, err := strconv.ParseUint(matches[1], 10, 64)
			if err != nil {
				s.log.
					WithField("fileName", file.Name()).
					WithField("filePath", filePath).
					WithError(err).Error("Failed to parse slot from beacon bad block file name")

				continue
			}

			slot := phase0.Slot(slotI)

			blockRoot := matches[2]

			// Read the file into the `block` variable
			blockRaw, err := os.ReadFile(filePath)
			if err != nil {
				s.log.
					WithField("slot", slot).
					WithField("blockRoot", blockRoot).
					WithField("filePath", filePath).
					WithError(err).
					Error("Failed to read beacon bad block file")

				continue
			}

			s.log.
				WithField("slot", slot).
				WithField("blockRoot", blockRoot).
				WithField("filePath", filePath).
				Debug("Processing beacon bad block")

			location := CreateBeaconBadBlockFileName(
				s.Config.Name,
				string(s.node.Beacon().Metadata().Network.Name),
				slot,
				blockRoot,
			)

			location = fmt.Sprintf("%s.ssz", location)

			exists := false

			// Check if we've somehow already indexed this beacon bad block
			rsp, err := s.indexer.ListBeaconBadBlock(ctx, &indexer.ListBeaconBadBlockRequest{
				Node:      s.Config.Name,
				BlockRoot: blockRoot,
				Slot:      slotI,
			})
			if err != nil {
				s.log.
					WithField("block_root", blockRoot).
					WithField("slot", slot).
					WithError(err).
					Error("Failed to check if beacon bad block is already indexed")
			}

			if rsp != nil && len(rsp.BeaconBadBlocks) > 0 {
				s.log.
					WithField("block_root", blockRoot).
					WithField("slot", slot).
					Debug("Beacon bad block already indexed")

				exists = true
			}

			if !exists {
				now := time.Now()

				s.log.WithField("location", location).Debug("Saving beacon bad block")

				location, err = s.store.SaveBeaconBadBlock(ctx, &blockRaw, location)
				if err != nil {
					s.log.WithFields(logrus.Fields{
						"slot":      slot,
						"blockRoot": blockRoot,
						"filePath":  filePath,
					}).WithError(err).Error("Failed to save beacon bad block to store")

					continue
				}

				// Sleep for 1s to give the store time to update
				time.Sleep(1 * time.Second)

				spec, err := s.node.Beacon().Node().Spec()
				if err != nil {
					s.log.
						WithField("slot", slot).
						WithField("blockRoot", blockRoot).
						WithField("filePath", filePath).
						WithError(err).Error("Failed to fetch spec")

					continue
				}

				req := &indexer.CreateBeaconBadBlockRequest{
					Node:        wrapperspb.String(s.Config.Name),
					Network:     wrapperspb.String(string(s.node.Beacon().Metadata().Network.Name)),
					Slot:        wrapperspb.UInt64(uint64(slot)),
					Epoch:       wrapperspb.UInt64(uint64(slot) / uint64(spec.SlotsPerEpoch)),
					BlockRoot:   wrapperspb.String(blockRoot),
					Location:    wrapperspb.String(location),
					NodeVersion: wrapperspb.String(s.node.Beacon().Metadata().NodeVersion(ctx)),
					BeaconImplementation: wrapperspb.String(
						s.node.Beacon().Metadata().Client(ctx),
					),
					FetchedAt: timestamppb.New(now),
				}

				// Index the block
				if _, err := s.indexer.CreateBeaconBadBlock(ctx, req); err != nil {
					s.log.
						WithField("block_root", blockRoot).
						WithField("slot", slot).
						WithError(err).
						Error("Failed to index beacon bad block")

					continue
				}

				s.metrics.IncrementItemExported(BeaconBadBlockQueue)

				s.log.
					WithField("block_root", blockRoot).
					WithField("slot", slot).
					Debug("Indexed beacon bad block")
			}

			// Delete the file
			if err := os.Remove(filePath); err != nil {
				s.log.
					WithField("filePath", filePath).
					WithError(err).
					Error("Failed to delete beacon bad block")

				continue
			}

			s.log.WithField("filePath", filePath).Debug("Deleted beacon bad block")
		}
	}

	return nil
}
