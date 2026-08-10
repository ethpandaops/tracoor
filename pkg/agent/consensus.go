package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	network := string(s.node.Beacon().Metadata().Network.Name)

	// Check if we've somehow already indexed this beacon state
	rsp, err := s.indexer.ListBeaconState(ctx, &indexer.ListBeaconStateRequest{
		Node:      s.Config.Name,
		StateRoot: rootAsString,
		Slot:      uint64(slot),
		Network:   network,
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

	spec, err := s.node.Beacon().Node().Spec()
	if err != nil {
		return err
	}

	target := &dedupTarget{
		kind:       store.BeaconStateDataType,
		queue:      BeaconStateQueue,
		network:    network,
		dedupKey:   beaconStateDedupKey(slot, rootAsString),
		directory:  BeaconStateDirectory(network, slot),
		identity:   rootAsString,
		extension:  ".ssz",
		slot:       uint64(slot),
		identifier: rootAsString,
		severity:   divergenceSeverityAlarm,
		save:       s.store.SaveBeaconState,
		remove:     s.store.DeleteBeaconState,
	}

	// States are the largest artifact the agent handles, so nothing about them
	// is ever held whole in memory: each read goes straight from the node into
	// a hash, and into the compressor as well when this agent is the one
	// storing it.
	fetcher := &streamFetcher{
		open: func(ctx context.Context) (*payloadSource, error) {
			raw, ferr := s.node.Beacon().Node().OpenRawBeaconState(ctx, stateID, string(mime.ContentTypeOctet))
			if ferr != nil {
				return nil, errors.Wrap(ferr, "failed to open beacon state")
			}

			return sourceFromResponse(raw), nil
		},
		compressor: s.compressor,
		save:       s.store.SaveBeaconState,
	}

	return s.indexDeduplicated(ctx, target, fetcher, func(ctx context.Context, outcome *dedupOutcome) error {
		req := &indexer.CreateBeaconStateRequest{
			Node:            wrapperspb.String(s.Config.Name),
			Network:         wrapperspb.String(network),
			Slot:            wrapperspb.UInt64(uint64(slot)),
			Epoch:           wrapperspb.UInt64(uint64(slot) / uint64(spec.SlotsPerEpoch)),
			StateRoot:       wrapperspb.String(rootAsString),
			Location:        wrapperspb.String(outcome.Location),
			ContentEncoding: wrapperspb.String(compression.Default.ContentEncoding),
			NodeVersion:     wrapperspb.String(s.node.Beacon().Metadata().NodeVersion(ctx)),
			BeaconImplementation: wrapperspb.String(
				s.node.Beacon().Metadata().Client(ctx),
			),
			FetchedAt:        timestamppb.New(now),
			ContentHash:      wrapperspb.String(outcome.ContentHash),
			VerifiedAt:       timestamppb.New(outcome.VerifiedAt),
			ContentMatchedAt: contentMatchedAt(outcome),
			DedupKey:         wrapperspb.String(target.dedupKey),
		}

		_, err := s.indexer.CreateBeaconState(ctx, req)

		return err
	})
}

func (s *agent) fetchAndIndexBeaconBlock(ctx context.Context, slot phase0.Slot) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	blockRoot, err := s.node.Beacon().Node().FetchBlockRoot(ctx, fmt.Sprintf("%d", slot))
	if err != nil {
		return errors.Wrap(err, "failed to fetch beacon block root")
	}

	blockRootAsString := blockRoot.String()

	network := string(s.node.Beacon().Metadata().Network.Name)

	// Check if we've somehow already indexed this beacon state
	rsp, err := s.indexer.ListBeaconBlock(ctx, &indexer.ListBeaconBlockRequest{
		Node:      s.Config.Name,
		BlockRoot: blockRootAsString,
		Slot:      uint64(slot),
		Network:   network,
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

	// Fetch by root rather than by slot: a reorg between resolving the root and
	// fetching the body would otherwise store one block's bytes under another's
	// root.
	blockID := blockRootAsString

	spec, err := s.node.Beacon().Node().Spec()
	if err != nil {
		return err
	}

	target := &dedupTarget{
		kind:       store.BeaconBlockDataType,
		queue:      BeaconBlockQueue,
		network:    network,
		dedupKey:   beaconBlockDedupKey(slot, blockRootAsString),
		directory:  BeaconBlockDirectory(network, slot),
		identity:   blockRootAsString,
		extension:  ".ssz",
		slot:       uint64(slot),
		identifier: blockRootAsString,
		severity:   divergenceSeverityAlarm,
		save:       s.store.SaveBeaconBlock,
		remove:     s.store.DeleteBeaconBlock,
	}

	fetcher := &streamFetcher{
		open: func(ctx context.Context) (*payloadSource, error) {
			raw, ferr := s.node.Beacon().Node().OpenRawBlock(ctx, blockID, string(mime.ContentTypeOctet))
			if ferr != nil {
				return nil, errors.Wrap(ferr, "failed to open beacon block")
			}

			return sourceFromResponse(raw), nil
		},
		compressor: s.compressor,
		save:       s.store.SaveBeaconBlock,
	}

	return s.indexDeduplicated(ctx, target, fetcher, func(ctx context.Context, outcome *dedupOutcome) error {
		req := &indexer.CreateBeaconBlockRequest{
			Node:            wrapperspb.String(s.Config.Name),
			Network:         wrapperspb.String(network),
			Slot:            wrapperspb.UInt64(uint64(slot)),
			Epoch:           wrapperspb.UInt64(uint64(slot) / uint64(spec.SlotsPerEpoch)),
			BlockRoot:       wrapperspb.String(blockRootAsString),
			Location:        wrapperspb.String(outcome.Location),
			ContentEncoding: wrapperspb.String(compression.Default.ContentEncoding),
			NodeVersion:     wrapperspb.String(s.node.Beacon().Metadata().NodeVersion(ctx)),
			BeaconImplementation: wrapperspb.String(
				s.node.Beacon().Metadata().Client(ctx),
			),
			FetchedAt:        timestamppb.New(now),
			ContentHash:      wrapperspb.String(outcome.ContentHash),
			VerifiedAt:       timestamppb.New(outcome.VerifiedAt),
			ContentMatchedAt: contentMatchedAt(outcome),
			DedupKey:         wrapperspb.String(target.dedupKey),
		}

		_, err := s.indexer.CreateBeaconBlock(ctx, req)

		return err
	})
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

// fetchAndIndexExecutionPayloadEnvelope archives the signed execution payload
// envelope for the block at the given slot. Envelopes exist from the gloas
// fork onwards and are revealed by the builder after the block arrives, so a
// not-found response means the payload has not (yet) been revealed. A payload
// that is never revealed leaves the slot without an envelope, which is not an
// error worth chasing.
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

	network := string(s.node.Beacon().Metadata().Network.Name)

	// Check if we've somehow already indexed this envelope
	rsp, err := s.indexer.ListExecutionPayloadEnvelope(ctx, &indexer.ListExecutionPayloadEnvelopeRequest{
		Node:      s.Config.Name,
		BlockRoot: blockRootAsString,
		Slot:      uint64(slot),
		Network:   network,
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

	target := &dedupTarget{
		kind:       store.ExecutionPayloadEnvelopeDataType,
		queue:      ExecutionPayloadEnvelopeQueue,
		network:    network,
		dedupKey:   executionPayloadEnvelopeDedupKey(slot, blockRootAsString),
		directory:  ExecutionPayloadEnvelopeDirectory(network, slot),
		identity:   blockRootAsString,
		extension:  ".ssz",
		slot:       uint64(slot),
		identifier: blockRootAsString,
		severity:   divergenceSeverityAlarm,
		save:       s.store.SaveExecutionPayloadEnvelope,
		remove:     s.store.DeleteExecutionPayloadEnvelope,
	}

	fetcher := &streamFetcher{
		open: func(ctx context.Context) (*payloadSource, error) {
			raw, ferr := s.node.Beacon().Node().OpenRawExecutionPayloadEnvelope(ctx, blockRootAsString, string(mime.ContentTypeOctet))
			if ferr != nil {
				if goerrors.Is(ferr, api.ErrNotFound) {
					return nil, fmt.Errorf("%w: execution payload envelope for %s has not been revealed", errItemNotAvailable, blockRootAsString)
				}

				return nil, errors.Wrap(ferr, "failed to fetch execution payload envelope")
			}

			return sourceFromResponse(raw), nil
		},
		compressor: s.compressor,
		save:       s.store.SaveExecutionPayloadEnvelope,
	}

	return s.indexDeduplicated(ctx, target, fetcher, func(ctx context.Context, outcome *dedupOutcome) error {
		req := &indexer.CreateExecutionPayloadEnvelopeRequest{
			Node:            wrapperspb.String(s.Config.Name),
			Network:         wrapperspb.String(network),
			Slot:            wrapperspb.UInt64(uint64(slot)),
			Epoch:           wrapperspb.UInt64(epoch),
			BlockRoot:       wrapperspb.String(blockRootAsString),
			Location:        wrapperspb.String(outcome.Location),
			ContentEncoding: wrapperspb.String(compression.Default.ContentEncoding),
			NodeVersion:     wrapperspb.String(s.node.Beacon().Metadata().NodeVersion(ctx)),
			BeaconImplementation: wrapperspb.String(
				s.node.Beacon().Metadata().Client(ctx),
			),
			FetchedAt:        timestamppb.New(now),
			ContentHash:      wrapperspb.String(outcome.ContentHash),
			VerifiedAt:       timestamppb.New(outcome.VerifiedAt),
			ContentMatchedAt: contentMatchedAt(outcome),
			DedupKey:         wrapperspb.String(target.dedupKey),
		}

		_, err := s.indexer.CreateExecutionPayloadEnvelope(ctx, req)

		return err
	})
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

				// Bad blocks are node-local by nature: one node rejecting a
				// block says nothing about what another holds under the same
				// root. They are hashed anyway, so two nodes that did happen to
				// keep identical bytes can be noticed.
				contentHash := sha256.Sum256(blockRaw)

				s.metrics.IncrementPayloadVerified(BeaconBadBlockQueue, s.Config.Name)

				compressedBlock, err := s.compressor.Compress(&blockRaw, compression.Default)
				if err != nil {
					return errors.Wrap(err, "failed to compress beacon bad block")
				}

				s.log.WithField("location", location).Debug("Saving beacon bad block")

				location, err = s.store.SaveBeaconBadBlock(ctx, &store.SaveParams{
					Data:            bytes.NewReader(compressedBlock),
					Location:        location,
					ContentEncoding: compression.Default.ContentEncoding,
				})
				if err != nil {
					s.log.WithFields(logrus.Fields{
						logKeySlot:  slot,
						"blockRoot": blockRoot,
						"filePath":  filePath,
					}).WithError(err).Error("Failed to save beacon bad block to store")

					continue
				}

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
					ContentEncoding: wrapperspb.String(compression.Default.ContentEncoding),
					NodeVersion:     wrapperspb.String(s.node.Beacon().Metadata().NodeVersion(ctx)),
					BeaconImplementation: wrapperspb.String(
						s.node.Beacon().Metadata().Client(ctx),
					),
					FetchedAt:   timestamppb.New(now),
					ContentHash: wrapperspb.String(hex.EncodeToString(contentHash[:])),
					VerifiedAt:  timestamppb.New(now),
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

				// Node-local like bad blocks, and hashed for the same reason.
				contentHash := sha256.Sum256(blobRaw)

				s.metrics.IncrementPayloadVerified(BeaconBadBlobQueue, s.Config.Name)

				// Compress it
				compressedBlob, err := s.compressor.Compress(&blobRaw, compression.Default)
				if err != nil {
					return errors.Wrap(err, "failed to compress beacon bad block")
				}

				s.log.WithField("location", location).Debug("Saving beacon bad blob")

				location, err = s.store.SaveBeaconBadBlob(ctx, &store.SaveParams{
					Data:            bytes.NewReader(compressedBlob),
					Location:        location,
					ContentEncoding: compression.Default.ContentEncoding,
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
					ContentEncoding: wrapperspb.String(compression.Default.ContentEncoding),
					NodeVersion:     wrapperspb.String(s.node.Beacon().Metadata().NodeVersion(ctx)),
					BeaconImplementation: wrapperspb.String(
						s.node.Beacon().Metadata().Client(ctx),
					),
					FetchedAt:   timestamppb.New(now),
					ContentHash: wrapperspb.String(hex.EncodeToString(contentHash[:])),
					VerifiedAt:  timestamppb.New(now),
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
