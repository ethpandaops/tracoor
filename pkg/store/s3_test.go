package store

import (
	"context"
	"testing"

	"bytes"

	"github.com/ethpandaops/tracoor/pkg/compression"
)

func TestS3StoreOperations(t *testing.T) {
	bucket := "mybucket"
	ctx := context.Background()

	store, cleanup, err := NewMockS3Store(ctx, bucket)
	if err != nil {
		t.Fatalf("Failed to create S3 store: %v", err)
	}

	defer func() {
		if err = cleanup(); err != nil {
			t.Fatalf("Failed to clean up: %v", err)
		}
	}()

	t.Run("BeaconState", func(t *testing.T) {
		testBeaconState(ctx, t, store)
	})

	t.Run("BeaconBlock", func(t *testing.T) {
		testBeaconBlock(ctx, t, store)
	})

	t.Run("BeaconBadBlock", func(t *testing.T) {
		testBeaconBadBlock(ctx, t, store)
	})

	t.Run("ExecutionBlockTrace", func(t *testing.T) {
		testExecutionBlockTrace(ctx, t, store)
	})

	t.Run("ExecutionBadBlock", func(t *testing.T) {
		testExecutionBadBlock(ctx, t, store)
	})

	t.Run("Copy", func(t *testing.T) {
		testCopy(ctx, t, store)
	})
}

func testBeaconState(ctx context.Context, t *testing.T, store Store) {
	t.Helper()

	location := "beacon_state/location.json"
	data := []byte(`"abc": "def"`)

	var err error

	compressor := compression.NewCompressor()

	t.Run("BeaconState", func(t *testing.T) {
		if err = store.Healthy(ctx); err != nil {
			t.Fatalf("Store is not healthy: %v", err)
		}

		compressedData, err := compressor.Compress(&data, compression.Gzip)
		if err != nil {
			t.Fatalf("Failed to compress data: %v", err)
		}

		location, err = store.SaveBeaconState(ctx, &SaveParams{
			Data:            &compressedData,
			Location:        location,
			ContentEncoding: compression.Gzip.ContentEncoding,
		})
		if err != nil {
			t.Fatalf("Failed to save beacon state: %v", err)
		}

		retrievedData, err := store.GetBeaconState(ctx, location)
		if err != nil {
			t.Fatalf("Failed to get beacon state: %v", err)
		}

		if retrievedData == nil {
			t.Fatal("Retrieved data is nil")
		}

		exists, err := store.Exists(ctx, location)
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}

		if !exists {
			t.Fatal("Expected file to exist")
		}

		if err = store.DeleteBeaconState(ctx, location); err != nil {
			t.Fatalf("Failed to delete beacon state: %v", err)
		}

		exists, err = store.Exists(ctx, location)
		if err != nil {
			t.Fatalf("Failed to check existence after deletion: %v", err)
		}

		if exists {
			t.Fatal("Expected file to not exist after deletion")
		}
	})
}

func testBeaconBlock(ctx context.Context, t *testing.T, store Store) {
	t.Helper()

	location := "beacon_block/location.json"
	data := []byte(`"abc": "def"`)

	var err error

	t.Run("BeaconBlock", func(t *testing.T) {
		if err = store.Healthy(ctx); err != nil {
			t.Fatalf("Store is not healthy: %v", err)
		}

		location, err = store.SaveBeaconBlock(ctx, &SaveParams{
			Data:            &data,
			Location:        location,
			ContentEncoding: "",
		})
		if err != nil {
			t.Fatalf("Failed to save beacon block: %v", err)
		}

		retrievedData, err := store.GetBeaconBlock(ctx, location)
		if err != nil {
			t.Fatalf("Failed to get beacon block: %v", err)
		}

		if retrievedData == nil {
			t.Fatal("Retrieved data is nil")
		}

		exists, err := store.Exists(ctx, location)
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}

		if !exists {
			t.Fatal("Expected file to exist")
		}

		if err = store.DeleteBeaconBlock(ctx, location); err != nil {
			t.Fatalf("Failed to delete beacon block: %v", err)
		}

		exists, err = store.Exists(ctx, location)
		if err != nil {
			t.Fatalf("Failed to check existence after deletion: %v", err)
		}

		if exists {
			t.Fatal("Expected file to not exist after deletion")
		}
	})
}

func testBeaconBadBlock(ctx context.Context, t *testing.T, store Store) {
	t.Helper()

	location := "beacon_bad_block/location.json"
	data := []byte(`"abc": "def"`)

	var err error

	t.Run("BeaconBadBlock", func(t *testing.T) {
		if err = store.Healthy(ctx); err != nil {
			t.Fatalf("Store is not healthy: %v", err)
		}

		location, err = store.SaveBeaconBadBlock(ctx, &SaveParams{
			Data:            &data,
			Location:        location,
			ContentEncoding: "",
		})
		if err != nil {
			t.Fatalf("Failed to save beacon bad block: %v", err)
		}

		retrievedData, err := store.GetBeaconBadBlock(ctx, location)
		if err != nil {
			t.Fatalf("Failed to get beacon bad block: %v", err)
		}

		if retrievedData == nil {
			t.Fatal("Retrieved data is nil")
		}

		exists, err := store.Exists(ctx, location)
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}

		if !exists {
			t.Fatal("Expected file to exist")
		}

		if err = store.DeleteBeaconBadBlock(ctx, location); err != nil {
			t.Fatalf("Failed to delete beacon bad block: %v", err)
		}

		exists, err = store.Exists(ctx, location)
		if err != nil {
			t.Fatalf("Failed to check existence after deletion: %v", err)
		}

		if exists {
			t.Fatal("Expected file to not exist after deletion")
		}
	})
}

func testExecutionBlockTrace(ctx context.Context, t *testing.T, store Store) {
	t.Helper()

	location := "execution_block_trace/location.json"
	data := []byte(`"abc": "def"`)

	var err error

	t.Run("ExecutionBlockTrace", func(t *testing.T) {
		if err = store.Healthy(ctx); err != nil {
			t.Fatalf("Store is not healthy: %v", err)
		}

		location, err = store.SaveExecutionBlockTrace(ctx, &SaveParams{
			Data:            &data,
			Location:        location,
			ContentEncoding: "",
		})
		if err != nil {
			t.Fatalf("Failed to save execution block trace: %v", err)
		}

		retrievedData, err := store.GetExecutionBlockTrace(ctx, location)
		if err != nil {
			t.Fatalf("Failed to get execution block trace: %v", err)
		}

		if retrievedData == nil {
			t.Fatal("Retrieved data is nil")
		}

		exists, err := store.Exists(ctx, location)
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}

		if !exists {
			t.Fatal("Expected file to exist")
		}

		if err = store.DeleteExecutionBlockTrace(ctx, location); err != nil {
			t.Fatalf("Failed to delete execution block trace: %v", err)
		}

		exists, err = store.Exists(ctx, location)
		if err != nil {
			t.Fatalf("Failed to check existence after deletion: %v", err)
		}

		if exists {
			t.Fatal("Expected file to not exist after deletion")
		}
	})
}

func testExecutionBadBlock(ctx context.Context, t *testing.T, store Store) {
	t.Helper()

	location := "execution_bad_block/location.json"
	data := []byte(`"abc": "def"`)

	var err error

	t.Run("ExecutionBadBlock", func(t *testing.T) {
		if err = store.Healthy(ctx); err != nil {
			t.Fatalf("Store is not healthy: %v", err)
		}

		location, err = store.SaveExecutionBadBlock(ctx, &SaveParams{
			Data:            &data,
			Location:        location,
			ContentEncoding: "",
		})
		if err != nil {
			t.Fatalf("Failed to save execution bad block: %v", err)
		}

		retrievedData, err := store.GetExecutionBadBlock(ctx, location)
		if err != nil {
			t.Fatalf("Failed to get execution bad block: %v", err)
		}

		if retrievedData == nil {
			t.Fatal("Retrieved data is nil")
		}

		exists, err := store.Exists(ctx, location)
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}

		if !exists {
			t.Fatal("Expected file to exist")
		}

		if err = store.DeleteExecutionBadBlock(ctx, location); err != nil {
			t.Fatalf("Failed to delete execution bad block: %v", err)
		}

		exists, err = store.Exists(ctx, location)
		if err != nil {
			t.Fatalf("Failed to check existence after deletion: %v", err)
		}

		if exists {
			t.Fatal("Expected file to not exist after deletion")
		}
	})
}

func testCopy(ctx context.Context, t *testing.T, store Store) {
	t.Helper()

	t.Run("Copy", func(t *testing.T) {
		if err := store.Healthy(ctx); err != nil {
			t.Fatalf("Store is not healthy: %v", err)
		}

		// Create a source file first
		sourceLocation := "beacon_block/location.json"
		sourceData := []byte(`{"test": "data"}`)

		_, err := store.SaveBeaconBlock(ctx, &SaveParams{
			Data:            &sourceData,
			Location:        sourceLocation,
			ContentEncoding: "",
		})
		if err != nil {
			t.Fatalf("Failed to save source file: %v", err)
		}

		// Test copying the file
		destLocation := "beacon_block/location_copy.json"

		err = store.Copy(ctx, &CopyParams{
			Source:      sourceLocation,
			Destination: destLocation,
		})
		if err != nil {
			t.Fatalf("Failed to copy file: %v", err)
		}

		// Verify source still exists
		sourceExists, err := store.Exists(ctx, sourceLocation)
		if err != nil {
			t.Fatalf("Failed to check source existence: %v", err)
		}

		if !sourceExists {
			t.Fatal("Source file should still exist after copy")
		}

		// Verify destination exists
		destExists, err := store.Exists(ctx, destLocation)
		if err != nil {
			t.Fatalf("Failed to check destination existence: %v", err)
		}

		if !destExists {
			t.Fatal("Destination file should exist after copy")
		}

		// Verify content is the same
		destData, err := store.GetBeaconBlock(ctx, destLocation)
		if err != nil {
			t.Fatalf("Failed to get destination data: %v", err)
		}

		if !bytes.Equal(*destData, sourceData) {
			t.Fatalf("Destination data doesn't match source")
		}
	})
}
