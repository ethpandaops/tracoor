package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
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
		_, err := fsStore.SaveBeaconState(ctx, &data, location)
		require.NoError(t, err)

		savedData, err := fsStore.GetBeaconState(ctx, location)
		require.NoError(t, err)
		require.Equal(t, data, *savedData)
	})

	t.Run("DeleteBeaconState", func(t *testing.T) {
		location := "beacon_state/location.json"
		data := []byte(`{"abc": "def"}`)
		_, err := fsStore.SaveBeaconState(ctx, &data, location)
		require.NoError(t, err)

		err = fsStore.DeleteBeaconState(ctx, location)
		require.NoError(t, err)

		exists, err := fsStore.Exists(ctx, location)
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("SaveBeaconBlock", func(t *testing.T) {
		location := "beacon_block/location.json"
		data := []byte(`{"block": "data"}`)
		_, err := fsStore.SaveBeaconBlock(ctx, &data, location)
		require.NoError(t, err)

		savedData, err := fsStore.GetBeaconBlock(ctx, location)
		require.NoError(t, err)
		require.Equal(t, data, *savedData)
	})

	t.Run("DeleteBeaconBlock", func(t *testing.T) {
		location := "beacon_block/location.json"
		data := []byte(`{"block": "data"}`)
		_, err := fsStore.SaveBeaconBlock(ctx, &data, location)
		require.NoError(t, err)

		err = fsStore.DeleteBeaconBlock(ctx, location)
		require.NoError(t, err)

		exists, err := fsStore.Exists(ctx, location)
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("SaveBeaconBadBlock", func(t *testing.T) {
		location := "beacon_bad_block/location.json"
		data := []byte(`{"bad_block": "data"}`)
		_, err := fsStore.SaveBeaconBadBlock(ctx, &data, location)
		require.NoError(t, err)

		savedData, err := fsStore.GetBeaconBadBlock(ctx, location)
		require.NoError(t, err)
		require.Equal(t, data, *savedData)
	})

	t.Run("DeleteBeaconBadBlock", func(t *testing.T) {
		location := "beacon_bad_block/location.json"
		data := []byte(`{"bad_block": "data"}`)
		_, err := fsStore.SaveBeaconBadBlock(ctx, &data, location)
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
		_, err := fsStore.SaveBeaconBadBlob(ctx, &data, location)
		require.NoError(t, err)

		savedData, err := fsStore.GetBeaconBadBlob(ctx, location)
		require.NoError(t, err)
		require.Equal(t, data, *savedData)
	})

	t.Run("DeleteBeaconBadBlob", func(t *testing.T) {
		location := "beacon_bad_blob/location.json"
		data := []byte(`{"bad_blob": "data"}`)
		_, err := fsStore.SaveBeaconBadBlob(ctx, &data, location)
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
		_, err := fsStore.SaveExecutionBlockTrace(ctx, &data, location)
		require.NoError(t, err)

		savedData, err := fsStore.GetExecutionBlockTrace(ctx, location)
		require.NoError(t, err)
		require.Equal(t, data, *savedData)
	})

	t.Run("DeleteExecutionBlockTrace", func(t *testing.T) {
		location := "execution_block_trace/location.json"
		data := []byte(`{"trace": "data"}`)
		_, err := fsStore.SaveExecutionBlockTrace(ctx, &data, location)
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
		_, err := fsStore.SaveExecutionBadBlock(ctx, &data, location)
		require.NoError(t, err)

		savedData, err := fsStore.GetExecutionBadBlock(ctx, location)
		require.NoError(t, err)
		require.Equal(t, data, *savedData)
	})

	t.Run("DeleteExecutionBadBlock", func(t *testing.T) {
		location := "execution_bad_block/location.json"
		data := []byte(`{"bad_block": "data"}`)
		_, err := fsStore.SaveExecutionBadBlock(ctx, &data, location)
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
}
