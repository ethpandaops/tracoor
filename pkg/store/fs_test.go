package store_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const (
	locationBadBeaconBlock = "beacon_bad_block/location.json"
	locationBeaconBlock    = "beacon_block/location.json"
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
			Data:     &data,
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
			Data:     &data,
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
			Data:     &data,
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
			Data:     &data,
			Location: location,
		})
		require.NoError(t, err)

		err = fsStore.DeleteBeaconBlock(ctx, location)
		require.NoError(t, err)

		exists, err := fsStore.Exists(ctx, location)
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("SaveBeaconBadBlock", func(t *testing.T) {
		location := locationBadBeaconBlock
		data := []byte(`{"bad_block": "data"}`)
		_, err := fsStore.SaveBeaconBadBlock(ctx, &store.SaveParams{
			Data:     &data,
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
			Data:     &data,
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
			Data:     &data,
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
			Data:     &data,
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
			Data:     &data,
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
			Data:     &data,
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
			Data:     &data,
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
			Data:     &data,
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
			Data:     &data,
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

func TestFSStorePathTraversal(t *testing.T) {
	root, err := os.MkdirTemp("", "fsstore_traversal_test")
	require.NoError(t, err)

	defer os.RemoveAll(root)

	basePath := filepath.Join(root, "store_data")
	require.NoError(t, os.MkdirAll(basePath, 0o755))

	// A file outside basePath that the store must never be able to reach.
	secretPath := filepath.Join(root, "secret.txt")
	secretContents := []byte("must not be reachable through the store")
	require.NoError(t, os.WriteFile(secretPath, secretContents, 0o600))

	log := logrus.New()
	fsStore, err := store.NewFSStore("test", log, &store.FSStoreConfig{BasePath: basePath}, nil)
	require.NoError(t, err)

	ctx := context.Background()

	maliciousLocations := []string{
		"../secret.txt",
		"../../secret.txt",
		"beacon_state/../../secret.txt",
		"./../secret.txt",
	}

	for _, location := range maliciousLocations {
		t.Run("Exists/"+location, func(t *testing.T) {
			_, err := fsStore.Exists(ctx, location)
			require.Error(t, err)
			require.True(t, errors.Is(err, store.ErrInvalid))
		})

		t.Run("GetBeaconState/"+location, func(t *testing.T) {
			_, err := fsStore.GetBeaconState(ctx, location)
			require.Error(t, err)
			require.True(t, errors.Is(err, store.ErrInvalid))
		})

		t.Run("DeleteBeaconState/"+location, func(t *testing.T) {
			err := fsStore.DeleteBeaconState(ctx, location)
			require.Error(t, err)
			require.True(t, errors.Is(err, store.ErrInvalid))
		})

		t.Run("SaveBeaconState/"+location, func(t *testing.T) {
			data := []byte("attacker controlled")
			_, err := fsStore.SaveBeaconState(ctx, &store.SaveParams{
				Data:     &data,
				Location: location,
			})
			require.Error(t, err)
			require.True(t, errors.Is(err, store.ErrInvalid))
		})
	}

	// The secret file must be completely untouched by all of the above.
	contents, err := os.ReadFile(secretPath)
	require.NoError(t, err)
	require.Equal(t, secretContents, contents)

	t.Run("Copy rejects a traversing source", func(t *testing.T) {
		err := fsStore.Copy(ctx, &store.CopyParams{
			Source:      "../secret.txt",
			Destination: "beacon_block/copied.json",
		})
		require.Error(t, err)
		require.True(t, errors.Is(err, store.ErrInvalid))
	})

	t.Run("Copy rejects a traversing destination", func(t *testing.T) {
		data := []byte(`{"block": "data"}`)
		_, err := fsStore.SaveBeaconBlock(ctx, &store.SaveParams{
			Data:     &data,
			Location: locationBeaconBlock,
		})
		require.NoError(t, err)

		err = fsStore.Copy(ctx, &store.CopyParams{
			Source:      locationBeaconBlock,
			Destination: "../escaped.json",
		})
		require.Error(t, err)
		require.True(t, errors.Is(err, store.ErrInvalid))
	})

	t.Run("legitimate nested locations are unaffected", func(t *testing.T) {
		location := "node1/mainnet/beacon_state/slot-123-0xabc.ssz"
		data := []byte(`{"legitimate": "data"}`)

		_, err := fsStore.SaveBeaconState(ctx, &store.SaveParams{
			Data:     &data,
			Location: location,
		})
		require.NoError(t, err)

		saved, err := fsStore.GetBeaconState(ctx, location)
		require.NoError(t, err)
		require.Equal(t, data, *saved)

		exists, err := fsStore.Exists(ctx, location)
		require.NoError(t, err)
		require.True(t, exists)

		require.NoError(t, fsStore.DeleteBeaconState(ctx, location))
	})
}
