package agent

import (
	"context"
	goerrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/ethpandaops/beacon/pkg/beacon/api"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/beacon/services"
	"github.com/ethpandaops/tracoor/pkg/compression"
	"github.com/ethpandaops/tracoor/pkg/mime"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/store"
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
		Network:   string(s.node.Beacon().Metadata().Network.Name),
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

	// Held until the upload finishes, not just the fetch, as the raw state and its
	// compressed copy are both live until then.
	releaseFetchSlot, err := s.acquireFetchSlot(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to acquire fetch slot")
	}

	defer releaseFetchSlot()

	// Fetch the state
	state, err := s.node.Beacon().Node().FetchRawBeaconState(ctx, stateID, string(mime.ContentTypeOctet))
	if err != nil {
		return err
	}

	s.log.WithField("location", location).Debug("Saving beacon state")

	// Compress it
	compressedState, err := s.compressor.Compress(&state, compression.Gzip)
	if err != nil {
		return errors.Wrap(err, "failed to compress beacon state")
	}

	// Drop the raw state before the upload so only the compressed copy is held
	// for the duration of the store write.
	state = nil

	// Upload the state to the store
	location, err = s.store.SaveBeaconState(ctx, &store.SaveParams{
		Data:            &compressedState,
		Location:        location,
		ContentEncoding: compression.Gzip.ContentEncoding,
	})
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
		Node:            wrapperspb.String(s.Config.Name),
		Network:         wrapperspb.String(string(s.node.Beacon().Metadata().Network.Name)),
		Slot:            wrapperspb.UInt64(uint64(slot)),
		Epoch:           wrapperspb.UInt64(uint64(slot) / uint64(spec.SlotsPerEpoch)),
		StateRoot:       wrapperspb.String(rootAsString),
		Location:        wrapperspb.String(location),
		ContentEncoding: wrapperspb.String(compression.Gzip.ContentEncoding),
		NodeVersion:     wrapperspb.String(s.node.Beacon().Metadata().NodeVersion(ctx)),
		BeaconImplementation: wrapperspb.String(
			s.node.Beacon().Metadata().Client(ctx),
		),
		FetchedAt: timestamppb.New(now),
	}

	// Index the state
	if _, err := s.indexer.CreateBeaconState(ctx, req); err != nil {
		return err
	}

	s.metrics.IncrementItemExported(BeaconStateQueue, s.Config.Name)

	s.log.
		WithField("state_root", rootAsString).
		WithField("slot", slot).
		Debug("Indexed beacon state")

	return nil
}

func (s *agent) fetchAndIndexBeaconBlock(ctx context.Context, slot phase0.Slot) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	blockRoot, err := s.node.Beacon().Node().FetchBlockRoot(ctx, fmt.Sprintf("%d", slot))
	if err != nil {
		return errors.Wrap(err, "failed to fetch beacon block root")
	}

	blockRootAsString := blockRoot.String()

	location := CreateBeaconBlockFileName(
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
		Network:   string(s.node.Beacon().Metadata().Network.Name),
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

	// Held until the upload finishes, not just the fetch.
	releaseFetchSlot, err := s.acquireFetchSlot(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to acquire fetch slot")
	}

	defer releaseFetchSlot()

	// Fetch the block
	blockRaw, err := s.node.Beacon().Node().FetchRawBlock(ctx, stateID, string(mime.ContentTypeOctet))
	if err != nil {
		return err
	}

	// Compress it
	compressedBlock, err := s.compressor.Compress(&blockRaw, compression.Gzip)
	if err != nil {
		return errors.Wrap(err, "failed to compress beacon block")
	}

	// Drop the raw block before the upload so only the compressed copy is held
	// for the duration of the store write.
	blockRaw = nil

	s.log.WithField("location", location).Debug("Saving beacon block")

	// Upload the block to the store
	location, err = s.store.SaveBeaconBlock(ctx, &store.SaveParams{
		Data:            &compressedBlock,
		Location:        location,
		ContentEncoding: compression.Gzip.ContentEncoding,
	})
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
		Node:            wrapperspb.String(s.Config.Name),
		Network:         wrapperspb.String(string(s.node.Beacon().Metadata().Network.Name)),
		Slot:            wrapperspb.UInt64(uint64(slot)),
		Epoch:           wrapperspb.UInt64(uint64(slot) / uint64(spec.SlotsPerEpoch)),
		BlockRoot:       wrapperspb.String(blockRootAsString),
		Location:        wrapperspb.String(location),
		ContentEncoding: wrapperspb.String(compression.Gzip.ContentEncoding),
		NodeVersion:     wrapperspb.String(s.node.Beacon().Metadata().NodeVersion(ctx)),
		BeaconImplementation: wrapperspb.String(
			s.node.Beacon().Metadata().Client(ctx),
		),
		FetchedAt: timestamppb.New(now),
	}

	// Index the block
	if _, err := s.indexer.CreateBeaconBlock(ctx, req); err != nil {
		return err
	}

	s.metrics.IncrementItemExported(BeaconBlockQueue, s.Config.Name)

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
	case string(services.ClientPrysm):
		pattern = `^beacon_block_(\d+)\.ssz$`
	default:
		return nil, errors.New("client does not have bad blocks available")
	}

	return &pattern, nil
}

const (
	executionPayloadEnvelopeFetchAttempts = 5
	executionPayloadEnvelopeFetchDelay    = 3 * time.Second
)

// fetchAndIndexExecutionPayloadEnvelope archives the signed execution payload
// envelope for the block at the given slot. Envelopes exist from the gloas
// fork onwards and are revealed by the builder after the block arrives, so
// the fetch retries briefly while the beacon node reports it as not found. A
// payload that is never revealed leaves the slot without an envelope, which
// is not an error.
func (s *agent) fetchAndIndexExecutionPayloadEnvelope(ctx context.Context, slot phase0.Slot) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	spec, err := s.node.Beacon().Node().Spec()
	if err != nil {
		return err
	}

	epoch := uint64(slot) / uint64(spec.SlotsPerEpoch)

	gloas, err := spec.ForkEpochs.GetByName("gloas")
	if err != nil || uint64(gloas.Epoch) > epoch {
		// The network has no gloas fork scheduled, or the slot pre-dates it.
		return nil
	}

	blockRoot, err := s.node.Beacon().Node().FetchBlockRoot(ctx, fmt.Sprintf("%d", slot))
	if err != nil {
		return errors.Wrap(err, "failed to fetch beacon block root")
	}

	blockRootAsString := blockRoot.String()

	location := CreateExecutionPayloadEnvelopeFileName(
		s.Config.Name,
		string(s.node.Beacon().Metadata().Network.Name),
		slot,
		blockRootAsString,
	)

	location = fmt.Sprintf("%s.ssz", location)

	// Check if we've somehow already indexed this envelope
	rsp, err := s.indexer.ListExecutionPayloadEnvelope(ctx, &indexer.ListExecutionPayloadEnvelopeRequest{
		Node:      s.Config.Name,
		BlockRoot: blockRootAsString,
		Slot:      uint64(slot),
		Network:   string(s.node.Beacon().Metadata().Network.Name),
	})
	if err != nil {
		s.log.
			WithField("block_root", blockRootAsString).
			WithField("slot", slot).
			WithError(err).
			Error("Failed to check if execution payload envelope is already indexed")
	}

	if rsp != nil && len(rsp.ExecutionPayloadEnvelopes) > 0 {
		s.log.
			WithField("block_root", blockRootAsString).
			WithField("slot", slot).
			Debug("Execution payload envelope already indexed")

		return nil
	}

	now := time.Now()

	var envelopeRaw []byte

	// Held only while a fetch is in flight and, on success, until the upload
	// finishes. The retry delays below are spent without a slot so a payload that
	// is never revealed does not occupy one for the whole backoff.
	var releaseFetchSlot func()

	defer func() {
		if releaseFetchSlot != nil {
			releaseFetchSlot()
		}
	}()

	for attempt := 0; attempt < executionPayloadEnvelopeFetchAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(executionPayloadEnvelopeFetchDelay):
			}
		}

		release, aErr := s.acquireFetchSlot(ctx)
		if aErr != nil {
			return errors.Wrap(aErr, "failed to acquire fetch slot")
		}

		envelopeRaw, err = s.node.Beacon().FetchRawExecutionPayloadEnvelope(ctx, blockRootAsString, string(mime.ContentTypeOctet))
		if err == nil {
			releaseFetchSlot = release

			break
		}

		release()

		if !goerrors.Is(err, api.ErrNotFound) {
			return errors.Wrap(err, "failed to fetch execution payload envelope")
		}
	}

	if err != nil {
		s.log.
			WithField("block_root", blockRootAsString).
			WithField("slot", slot).
			Debug("Execution payload envelope not available, payload was likely never revealed")

		return nil
	}

	// Compress it
	compressedEnvelope, err := s.compressor.Compress(&envelopeRaw, compression.Gzip)
	if err != nil {
		return errors.Wrap(err, "failed to compress execution payload envelope")
	}

	// Drop the raw envelope before the upload so only the compressed copy is held
	// for the duration of the store write.
	envelopeRaw = nil

	s.log.WithField("location", location).Debug("Saving execution payload envelope")

	// Upload the envelope to the store
	location, err = s.store.SaveExecutionPayloadEnvelope(ctx, &store.SaveParams{
		Data:            &compressedEnvelope,
		Location:        location,
		ContentEncoding: compression.Gzip.ContentEncoding,
	})
	if err != nil {
		return err
	}

	// Sleep for 1s to give the store time to update
	time.Sleep(1 * time.Second)

	req := &indexer.CreateExecutionPayloadEnvelopeRequest{
		Node:            wrapperspb.String(s.Config.Name),
		Network:         wrapperspb.String(string(s.node.Beacon().Metadata().Network.Name)),
		Slot:            wrapperspb.UInt64(uint64(slot)),
		Epoch:           wrapperspb.UInt64(epoch),
		BlockRoot:       wrapperspb.String(blockRootAsString),
		Location:        wrapperspb.String(location),
		ContentEncoding: wrapperspb.String(compression.Gzip.ContentEncoding),
		NodeVersion:     wrapperspb.String(s.node.Beacon().Metadata().NodeVersion(ctx)),
		BeaconImplementation: wrapperspb.String(
			s.node.Beacon().Metadata().Client(ctx),
		),
		FetchedAt: timestamppb.New(now),
	}

	// Index the envelope
	if _, err := s.indexer.CreateExecutionPayloadEnvelope(ctx, req); err != nil {
		return err
	}

	s.metrics.IncrementItemExported(ExecutionPayloadEnvelopeQueue, s.Config.Name)

	s.log.
		WithField("block_root", blockRootAsString).
		WithField("slot", slot).
		Debug("Indexed execution payload envelope")

	return nil
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
		if len(matches) == 2 {
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

			blockRoot := "unknown"

			if len(matches) == 3 {
				blockRoot = matches[2]
			}

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
				Network:   string(s.node.Beacon().Metadata().Network.Name),
			})
			if err != nil {
				s.log.
					WithField("blockRoot", blockRoot).
					WithField("slot", slot).
					WithError(err).
					Error("Failed to check if beacon bad block is already indexed")
			}

			if rsp != nil && len(rsp.BeaconBadBlocks) > 0 {
				s.log.
					WithField("blockRoot", blockRoot).
					WithField("slot", slot).
					Debug("Beacon bad block already indexed")

				exists = true
			}

			if !exists {
				now := time.Now()

				compressedBlock, err := s.compressor.Compress(&blockRaw, compression.Gzip)
				if err != nil {
					return errors.Wrap(err, "failed to compress beacon bad block")
				}

				s.log.WithField("location", location).Debug("Saving beacon bad block")

				location, err = s.store.SaveBeaconBadBlock(ctx, &store.SaveParams{
					Data:            &compressedBlock,
					Location:        location,
					ContentEncoding: compression.Gzip.ContentEncoding,
				})
				if err != nil {
					s.log.WithFields(logrus.Fields{
						logKeySlot:  slot,
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
					Node:            wrapperspb.String(s.Config.Name),
					Network:         wrapperspb.String(string(s.node.Beacon().Metadata().Network.Name)),
					Slot:            wrapperspb.UInt64(uint64(slot)),
					Epoch:           wrapperspb.UInt64(uint64(slot) / uint64(spec.SlotsPerEpoch)),
					BlockRoot:       wrapperspb.String(blockRoot),
					Location:        wrapperspb.String(location),
					ContentEncoding: wrapperspb.String(compression.Gzip.ContentEncoding),
					NodeVersion:     wrapperspb.String(s.node.Beacon().Metadata().NodeVersion(ctx)),
					BeaconImplementation: wrapperspb.String(
						s.node.Beacon().Metadata().Client(ctx),
					),
					FetchedAt: timestamppb.New(now),
				}

				// Index the block
				if _, err := s.indexer.CreateBeaconBadBlock(ctx, req); err != nil {
					s.log.
						WithField("blockRoot", blockRoot).
						WithField("slot", slot).
						WithError(err).
						Error("Failed to index beacon bad block")

					continue
				}

				s.metrics.IncrementItemExported(BeaconBadBlockQueue, s.Config.Name)

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

func getBadBlobsFilePattern(client string) (*string, error) {
	var pattern string

	switch client {
	case string(services.ClientPrysm):
		pattern = `^blob_sidecar_([^.]+)_(\d+)_(\d+)\.ssz$`
	default:
		return nil, errors.New("client does not have bad blobs available")
	}

	return &pattern, nil
}

func (s *agent) fetchAndIndexBeaconBadBlobs(ctx context.Context, path string) error {
	client := s.node.Beacon().Metadata().Client(ctx)

	// Verify the path is a directory
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("path %s is not a directory", path)
	}

	pattern, err := getBadBlobsFilePattern(client)
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
		if len(matches) == 4 {
			filePath := filepath.Join(path, file.Name())
			// Parse 'slot', 'blockRoot' and 'index' from the file name
			blockRoot := matches[1]

			slotI, err := strconv.ParseUint(matches[2], 10, 64)
			if err != nil {
				s.log.
					WithField("fileName", file.Name()).
					WithField("filePath", filePath).
					WithError(err).Error("Failed to parse slot from beacon bad blob file name")

				continue
			}

			slot := phase0.Slot(slotI)

			index, err := strconv.ParseUint(matches[3], 10, 64)
			if err != nil {
				s.log.
					WithField("fileName", file.Name()).
					WithField("filePath", filePath).
					WithError(err).Error("Failed to parse index from beacon bad blob file name")

				continue
			}

			// Read the file into the `blob` variable
			blobRaw, err := os.ReadFile(filePath)
			if err != nil {
				s.log.
					WithField("slot", slot).
					WithField("blockRoot", blockRoot).
					WithField("index", index).
					WithField("filePath", filePath).
					WithError(err).
					Error("Failed to read beacon bad blob file")

				continue
			}

			s.log.
				WithField("slot", slot).
				WithField("blockRoot", blockRoot).
				WithField("index", index).
				WithField("filePath", filePath).
				Debug("Processing beacon bad block")

			location := CreateBeaconBadBlobFileName(
				s.Config.Name,
				string(s.node.Beacon().Metadata().Network.Name),
				slot,
				blockRoot,
				index,
			)

			location = fmt.Sprintf("%s.ssz", location)

			exists := false

			// Check if we've somehow already indexed this beacon bad blob
			rsp, err := s.indexer.ListBeaconBadBlob(ctx, &indexer.ListBeaconBadBlobRequest{
				Node:      s.Config.Name,
				BlockRoot: blockRoot,
				Slot:      slotI,
				Index:     wrapperspb.UInt64(index),
				Network:   string(s.node.Beacon().Metadata().Network.Name),
			})
			if err != nil {
				s.log.
					WithField("index", index).
					WithField("blockRoot", blockRoot).
					WithField("slot", slot).
					WithError(err).
					Error("Failed to check if beacon bad blob is already indexed")
			}

			if rsp != nil && len(rsp.BeaconBadBlobs) > 0 {
				s.log.
					WithField("index", index).
					WithField("blockRoot", blockRoot).
					WithField("slot", slot).
					Debug("Beacon bad blob already indexed")

				exists = true
			}

			if !exists {
				now := time.Now()

				// Compress it
				compressedBlob, err := s.compressor.Compress(&blobRaw, compression.Gzip)
				if err != nil {
					return errors.Wrap(err, "failed to compress beacon bad block")
				}

				s.log.WithField("location", location).Debug("Saving beacon bad blob")

				location, err = s.store.SaveBeaconBadBlob(ctx, &store.SaveParams{
					Data:            &compressedBlob,
					Location:        location,
					ContentEncoding: compression.Gzip.ContentEncoding,
				})
				if err != nil {
					s.log.WithFields(logrus.Fields{
						logKeySlot:  slot,
						"blockRoot": blockRoot,
						"index":     index,
						"filePath":  filePath,
					}).WithError(err).Error("Failed to save beacon bad blob to store")

					continue
				}

				// Sleep for 1s to give the store time to update
				time.Sleep(1 * time.Second)

				spec, err := s.node.Beacon().Node().Spec()
				if err != nil {
					s.log.
						WithField("slot", slot).
						WithField("blockRoot", blockRoot).
						WithField("index", index).
						WithField("filePath", filePath).
						WithError(err).Error("Failed to fetch spec")

					continue
				}

				req := &indexer.CreateBeaconBadBlobRequest{
					Node:            wrapperspb.String(s.Config.Name),
					Network:         wrapperspb.String(string(s.node.Beacon().Metadata().Network.Name)),
					Slot:            wrapperspb.UInt64(uint64(slot)),
					Epoch:           wrapperspb.UInt64(uint64(slot) / uint64(spec.SlotsPerEpoch)),
					BlockRoot:       wrapperspb.String(blockRoot),
					Index:           wrapperspb.UInt64(index),
					Location:        wrapperspb.String(location),
					ContentEncoding: wrapperspb.String(compression.Gzip.ContentEncoding),
					NodeVersion:     wrapperspb.String(s.node.Beacon().Metadata().NodeVersion(ctx)),
					BeaconImplementation: wrapperspb.String(
						s.node.Beacon().Metadata().Client(ctx),
					),
					FetchedAt: timestamppb.New(now),
				}

				// Index the blob
				if _, err := s.indexer.CreateBeaconBadBlob(ctx, req); err != nil {
					s.log.
						WithField("blockRoot", blockRoot).
						WithField("index", index).
						WithField("slot", slot).
						WithError(err).
						Error("Failed to index beacon bad blob")

					continue
				}

				s.metrics.IncrementItemExported(BeaconBadBlobQueue, s.Config.Name)

				s.log.
					WithField("blockRoot", blockRoot).
					WithField("index", index).
					WithField("slot", slot).
					Debug("Indexed beacon bad block")
			}

			// Delete the file
			if err := os.Remove(filePath); err != nil {
				s.log.
					WithField("filePath", filePath).
					WithError(err).
					Error("Failed to delete beacon bad blob")

				continue
			}

			s.log.WithField("filePath", filePath).Debug("Deleted beacon bad blob")
		}
	}

	return nil
}
