package store_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const (
	locationBadBeaconBlock           = "beacon_bad_block/location.json"
	locationBeaconBlock              = "beacon_block/location.json"
	locationExecutionPayloadEnvelope = "execution_payload_envelope/location.json"
)

func TestFSStoreOperations(t *testing.T) {
	basePath, err := os.MkdirTemp("", "fsstore_test")
	require.NoError(t, err)

	defer os.RemoveAll(basePath)

	log := logrus.New()
	fsStore, err := store.NewFSStore("test", log, &store.FSStoreConfig{BasePath: basePath}, nil)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Healthy", func(t *testing.T) {
		err := fsStore.Healthy(ctx)
		require.NoError(t, err)
	})

	t.Run("SaveBeaconState", func(t *testing.T) {
		location := "beacon_state/location.json"
		data := []byte(`{"abc": "def"}`)
		_, err := fsStore.SaveBeaconState(ctx, &store.SaveParams{
			Data:     bytes.NewReader(data),
			Location: location,
		})
		require.NoError(t, err)

		savedData, err := fsStore.GetBeaconState(ctx, location)
		require.NoError(t, err)
		require.Equal(t, data, *savedData)
	})

	t.Run("DeleteBeaconState", func(t *testing.T) {
		location := "beacon_state/location.json"
		data := []byte(`{"abc": "def"}`)
		_, err := fsStore.SaveBeaconState(ctx, &store.SaveParams{
			Data:     bytes.NewReader(data),
			Location: location,
		})
		require.NoError(t, err)

		err = fsStore.DeleteBeaconState(ctx, location)
		require.NoError(t, err)

		exists, err := fsStore.Exists(ctx, location)
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("SaveBeaconBlock", func(t *testing.T) {
		location := locationBeaconBlock
		data := []byte(`{"block": "data"}`)
		_, err := fsStore.SaveBeaconBlock(ctx, &store.SaveParams{
			Data:     bytes.NewReader(data),
			Location: location,
		})
		require.NoError(t, err)

		savedData, err := fsStore.GetBeaconBlock(ctx, location)
		require.NoError(t, err)
		require.Equal(t, data, *savedData)
	})

	t.Run("DeleteBeaconBlock", func(t *testing.T) {
		location := locationBeaconBlock
		data := []byte(`{"block": "data"}`)
		_, err := fsStore.SaveBeaconBlock(ctx, &store.SaveParams{
			Data:     bytes.NewReader(data),
			Location: location,
		})
		require.NoError(t, err)

		err = fsStore.DeleteBeaconBlock(ctx, location)
		require.NoError(t, err)

		exists, err := fsStore.Exists(ctx, location)
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("SaveExecutionPayloadEnvelope", func(t *testing.T) {
		location := locationExecutionPayloadEnvelope
		data := []byte(`{"block": "data"}`)
		_, err := fsStore.SaveExecutionPayloadEnvelope(ctx, &store.SaveParams{
			Data:     bytes.NewReader(data),
			Location: location,
		})
		require.NoError(t, err)

		savedData, err := fsStore.GetExecutionPayloadEnvelope(ctx, location)
		require.NoError(t, err)
		require.Equal(t, data, *savedData)
	})

	t.Run("DeleteExecutionPayloadEnvelope", func(t *testing.T) {
		location := locationExecutionPayloadEnvelope
		data := []byte(`{"block": "data"}`)
		_, err := fsStore.SaveExecutionPayloadEnvelope(ctx, &store.SaveParams{
			Data:     bytes.NewReader(data),
			Location: location,
		})
		require.NoError(t, err)

		err = fsStore.DeleteExecutionPayloadEnvelope(ctx, location)
		require.NoError(t, err)

		exists, err := fsStore.Exists(ctx, location)
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("SaveBeaconBadBlock", func(t *testing.T) {
		location := locationBadBeaconBlock
		data := []byte(`{"bad_block": "data"}`)
		_, err := fsStore.SaveBeaconBadBlock(ctx, &store.SaveParams{
			Data:     bytes.NewReader(data),
			Location: location,
		})
		require.NoError(t, err)

		savedData, err := fsStore.GetBeaconBadBlock(ctx, location)
		require.NoError(t, err)
		require.Equal(t, data, *savedData)
	})

	t.Run("DeleteBeaconBadBlock", func(t *testing.T) {
		location := locationBadBeaconBlock
		data := []byte(`{"bad_block": "data"}`)
		_, err := fsStore.SaveBeaconBadBlock(ctx, &store.SaveParams{
			Data:     bytes.NewReader(data),
			Location: location,
		})
		require.NoError(t, err)

		err = fsStore.DeleteBeaconBadBlock(ctx, location)
		require.NoError(t, err)

		exists, err := fsStore.Exists(ctx, location)
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("SaveBeaconBadBlob", func(t *testing.T) {
		location := "beacon_bad_blob/location.json"
		data := []byte(`{"bad_blob": "data"}`)
		_, err := fsStore.SaveBeaconBadBlob(ctx, &store.SaveParams{
			Data:     bytes.NewReader(data),
			Location: location,
		})
		require.NoError(t, err)

		savedData, err := fsStore.GetBeaconBadBlob(ctx, location)
		require.NoError(t, err)
		require.Equal(t, data, *savedData)
	})

	t.Run("DeleteBeaconBadBlob", func(t *testing.T) {
		location := "beacon_bad_blob/location.json"
		data := []byte(`{"bad_blob": "data"}`)
		_, err := fsStore.SaveBeaconBadBlob(ctx, &store.SaveParams{
			Data:     bytes.NewReader(data),
			Location: location,
		})
		require.NoError(t, err)

		err = fsStore.DeleteBeaconBadBlob(ctx, location)
		require.NoError(t, err)

		exists, err := fsStore.Exists(ctx, location)
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("SaveExecutionBlockTrace", func(t *testing.T) {
		location := "execution_block_trace/location.json"
		data := []byte(`{"trace": "data"}`)
		_, err := fsStore.SaveExecutionBlockTrace(ctx, &store.SaveParams{
			Data:     bytes.NewReader(data),
			Location: location,
		})
		require.NoError(t, err)

		savedData, err := fsStore.GetExecutionBlockTrace(ctx, location)
		require.NoError(t, err)
		require.Equal(t, data, *savedData)
	})

	t.Run("DeleteExecutionBlockTrace", func(t *testing.T) {
		location := "execution_block_trace/location.json"
		data := []byte(`{"trace": "data"}`)
		_, err := fsStore.SaveExecutionBlockTrace(ctx, &store.SaveParams{
			Data:     bytes.NewReader(data),
			Location: location,
		})
		require.NoError(t, err)

		err = fsStore.DeleteExecutionBlockTrace(ctx, location)
		require.NoError(t, err)

		exists, err := fsStore.Exists(ctx, location)
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("SaveExecutionBadBlock", func(t *testing.T) {
		location := "execution_bad_block/location.json"
		data := []byte(`{"bad_block": "data"}`)
		_, err := fsStore.SaveExecutionBadBlock(ctx, &store.SaveParams{
			Data:     bytes.NewReader(data),
			Location: location,
		})
		require.NoError(t, err)

		savedData, err := fsStore.GetExecutionBadBlock(ctx, location)
		require.NoError(t, err)
		require.Equal(t, data, *savedData)
	})

	t.Run("DeleteExecutionBadBlock", func(t *testing.T) {
		location := "execution_bad_block/location.json"
		data := []byte(`{"bad_block": "data"}`)
		_, err := fsStore.SaveExecutionBadBlock(ctx, &store.SaveParams{
			Data:     bytes.NewReader(data),
			Location: location,
		})
		require.NoError(t, err)

		err = fsStore.DeleteExecutionBadBlock(ctx, location)
		require.NoError(t, err)

		exists, err := fsStore.Exists(ctx, location)
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("StorageHandshakeTokenExists", func(t *testing.T) {
		node := "node1"
		exists, err := fsStore.StorageHandshakeTokenExists(ctx, node)
		require.NoError(t, err)
		require.False(t, exists)

		err = fsStore.SaveStorageHandshakeToken(ctx, node, "token_data")
		require.NoError(t, err)

		exists, err = fsStore.StorageHandshakeTokenExists(ctx, node)
		require.NoError(t, err)
		require.True(t, exists)
	})

	t.Run("SaveStorageHandshakeToken", func(t *testing.T) {
		node := "node2"
		data := "handshake_token_data"
		err := fsStore.SaveStorageHandshakeToken(ctx, node, data)
		require.NoError(t, err)

		savedData, err := fsStore.GetStorageHandshakeToken(ctx, node)
		require.NoError(t, err)
		require.Equal(t, data, savedData)
	})

	t.Run("GetStorageHandshakeToken", func(t *testing.T) {
		node := "node3"
		data := "handshake_token_data"
		err := fsStore.SaveStorageHandshakeToken(ctx, node, data)
		require.NoError(t, err)

		savedData, err := fsStore.GetStorageHandshakeToken(ctx, node)
		require.NoError(t, err)
		require.Equal(t, data, savedData)
	})

	t.Run("Copy", func(t *testing.T) {
		location := locationBeaconBlock
		data := []byte(`{"block": "data"}`)
		_, err := fsStore.SaveBeaconBlock(ctx, &store.SaveParams{
			Data:     bytes.NewReader(data),
			Location: location,
		})
		require.NoError(t, err)

		err = fsStore.Copy(ctx, &store.CopyParams{
			Source:      location,
			Destination: "beacon_block/location_copy.json",
		})
		require.NoError(t, err)

		exists, err := fsStore.Exists(ctx, "beacon_block/location_copy.json")
		require.NoError(t, err)
		require.True(t, exists)
	})
}

// errAfterReader serves n bytes and then fails, standing in for a payload whose
// source dies part way through the transfer.
type errAfterReader struct {
	remaining int
	err       error
}

func (r *errAfterReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, r.err
	}

	if len(p) > r.remaining {
		p = p[:r.remaining]
	}

	for i := range p {
		p[i] = 'x'
	}

	r.remaining -= len(p)

	return len(p), nil
}

// visibleFiles lists the directory entries a reader of the store would see,
// which is every entry including any temporary file left behind.
func visibleFiles(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		require.NoError(t, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	return names
}

func TestFSStorePublishesAtomically(t *testing.T) {
	basePath, err := os.MkdirTemp("", "fsstore_atomic_test")
	require.NoError(t, err)

	defer os.RemoveAll(basePath)

	log := logrus.New()
	fsStore, err := store.NewFSStore("test", log, &store.FSStoreConfig{BasePath: basePath}, nil)
	require.NoError(t, err)

	ctx := context.Background()

	location := "beacon_state/atomic.ssz"
	dir := filepath.Join(basePath, "beacon_state")

	t.Run("FailedReadPublishesNothing", func(t *testing.T) {
		readErr := errors.New("source died")

		_, err := fsStore.SaveBeaconState(ctx, &store.SaveParams{
			Data:     &errAfterReader{remaining: 1024, err: readErr},
			Location: location,
		})
		require.ErrorIs(t, err, readErr)

		exists, err := fsStore.Exists(ctx, location)
		require.NoError(t, err)
		require.False(t, exists, "a failed save must not publish the location")

		require.Empty(t, visibleFiles(t, dir), "a failed save must not leave a temporary file behind")
	})

	t.Run("OverwriteKeepsThePreviousBytesUntilTheNewOnesAreComplete", func(t *testing.T) {
		original := []byte("original-payload")

		_, err := fsStore.SaveBeaconState(ctx, &store.SaveParams{
			Data:     bytes.NewReader(original),
			Location: location,
		})
		require.NoError(t, err)

		_, err = fsStore.SaveBeaconState(ctx, &store.SaveParams{
			Data:     io.MultiReader(strings.NewReader("partial"), &errAfterReader{err: errors.New("source died")}),
			Location: location,
		})
		require.Error(t, err)

		saved, err := fsStore.GetBeaconState(ctx, location)
		require.NoError(t, err)
		require.Equal(t, original, *saved, "a failed overwrite must leave the previous object intact")

		require.Equal(t, []string{"atomic.ssz"}, visibleFiles(t, dir))
	})

	t.Run("SuccessfulSaveLeavesOnlyTheFinalObject", func(t *testing.T) {
		payload := []byte("final-payload")

		_, err := fsStore.SaveBeaconState(ctx, &store.SaveParams{
			Data:     bytes.NewReader(payload),
			Location: location,
		})
		require.NoError(t, err)

		saved, err := fsStore.GetBeaconState(ctx, location)
		require.NoError(t, err)
		require.Equal(t, payload, *saved)

		require.Equal(t, []string{"atomic.ssz"}, visibleFiles(t, dir))
	})

	t.Run("NilDataIsRejected", func(t *testing.T) {
		_, err := fsStore.SaveBeaconState(ctx, &store.SaveParams{
			Location: "beacon_state/nil.ssz",
		})
		require.Error(t, err)
	})
}

func TestFSStoreDeleteMany(t *testing.T) {
	dir, err := os.MkdirTemp("", "fsstore_delete_many")
	require.NoError(t, err)

	defer os.RemoveAll(dir)

	fsStore, err := store.NewFSStore("test", logrus.New(), &store.FSStoreConfig{BasePath: dir}, nil)
	require.NoError(t, err)

	ctx := context.Background()

	locations := []string{
		"beacon_state/one.ssz",
		"beacon_state/two.ssz",
		"beacon_state/three.ssz",
	}

	for _, location := range locations {
		_, serr := fsStore.SaveBeaconState(ctx, &store.SaveParams{
			Data:     bytes.NewReader([]byte("payload")),
			Location: location,
		})
		require.NoError(t, serr)
	}

	t.Run("RemovesEveryLocation", func(t *testing.T) {
		require.NoError(t, fsStore.DeleteMany(ctx, locations))

		for _, location := range locations {
			exists, eerr := fsStore.Exists(ctx, location)
			require.NoError(t, eerr)
			require.False(t, exists)
		}
	})

	t.Run("AbsentLocationsAreNotAFailure", func(t *testing.T) {
		require.NoError(t, fsStore.DeleteMany(ctx, []string{"beacon_state/never-existed.ssz"}))
	})

	t.Run("EmptyInputIsANoOp", func(t *testing.T) {
		require.NoError(t, fsStore.DeleteMany(ctx, nil))
	})

	t.Run("ReportsTheLocationsItCouldNotRemove", func(t *testing.T) {
		// A directory in place of an object cannot be unlinked, which is the closest a
		// filesystem gets to a poisoned key.
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "beacon_state", "stuck.ssz", "child"), 0o755))

		_, serr := fsStore.SaveBeaconState(ctx, &store.SaveParams{
			Data:     bytes.NewReader([]byte("payload")),
			Location: "beacon_state/fine.ssz",
		})
		require.NoError(t, serr)

		derr := fsStore.DeleteMany(ctx, []string{"beacon_state/stuck.ssz", "beacon_state/fine.ssz"})
		require.Error(t, derr)

		var partial *store.DeleteManyError

		require.ErrorAs(t, derr, &partial)
		require.Equal(t, []string{"beacon_state/stuck.ssz"}, partial.Failed)

		exists, eerr := fsStore.Exists(ctx, "beacon_state/fine.ssz")
		require.NoError(t, eerr)
		require.False(t, exists, "a poisoned location must not stop the rest of the batch")
	})
}
