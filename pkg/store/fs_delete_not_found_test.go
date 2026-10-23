package store_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TestFSStoreDeleteMissingFileReturnsErrNotFound is a regression test for
// NM-W1-003: FSStore's Delete* methods used to return the raw *PathError
// from os.Remove when the target file didn't exist, rather than the
// package's store.ErrNotFound sentinel. Callers such as the retention
// watcher use errors.Is(err, store.ErrNotFound) to decide whether a missing
// file is fine to treat as "already gone" (and proceed to remove the
// database row) versus a real failure worth retrying. Since the FS backend
// never produced that sentinel, a missing file permanently blocked the
// corresponding database row from ever being cleaned up.
func TestFSStoreDeleteMissingFileReturnsErrNotFound(t *testing.T) {
	basePath, err := os.MkdirTemp("", "fsstore_delete_not_found_test")
	require.NoError(t, err)

	defer os.RemoveAll(basePath)

	fsStore, err := store.NewFSStore("test", logrus.New(), &store.FSStoreConfig{BasePath: basePath}, nil)
	require.NoError(t, err)

	ctx := context.Background()

	deleters := map[string]func(context.Context, string) error{
		"DeleteBeaconState":         fsStore.DeleteBeaconState,
		"DeleteBeaconBlock":         fsStore.DeleteBeaconBlock,
		"DeleteBeaconBadBlock":      fsStore.DeleteBeaconBadBlock,
		"DeleteBeaconBadBlob":       fsStore.DeleteBeaconBadBlob,
		"DeleteExecutionBlockTrace": fsStore.DeleteExecutionBlockTrace,
		"DeleteExecutionBadBlock":   fsStore.DeleteExecutionBadBlock,
	}

	for name, deleteFn := range deleters {
		t.Run(name, func(t *testing.T) {
			err := deleteFn(ctx, "does/not/exist.json")
			require.Error(t, err)
			require.True(t, errors.Is(err, store.ErrNotFound), "expected errors.Is(err, store.ErrNotFound) to be true, got: %v", err)
		})
	}
}

// TestFSStoreDeleteExistingFileStillSucceeds guards against the fix
// accidentally turning every delete into an error.
func TestFSStoreDeleteExistingFileStillSucceeds(t *testing.T) {
	basePath, err := os.MkdirTemp("", "fsstore_delete_existing_test")
	require.NoError(t, err)

	defer os.RemoveAll(basePath)

	fsStore, err := store.NewFSStore("test", logrus.New(), &store.FSStoreConfig{BasePath: basePath}, nil)
	require.NoError(t, err)

	ctx := context.Background()
	location := "beacon_state/present.json"
	data := []byte(`{"a":"b"}`)

	_, err = fsStore.SaveBeaconState(ctx, &store.SaveParams{Data: &data, Location: location})
	require.NoError(t, err)

	require.NoError(t, fsStore.DeleteBeaconState(ctx, location))

	exists, err := fsStore.Exists(ctx, location)
	require.NoError(t, err)
	require.False(t, exists)
}
