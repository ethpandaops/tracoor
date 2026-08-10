package store

import (
	"bytes"
	"context"
	"errors"
	"testing"

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

	t.Run("ExecutionPayloadEnvelope", func(t *testing.T) {
		testExecutionPayloadEnvelope(ctx, t, store)
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

	t.Run("AbortsOnAFailedRead", func(t *testing.T) {
		testAbortsOnAFailedRead(ctx, t, store)
	})

	t.Run("DeleteMany", func(t *testing.T) {
		testDeleteMany(ctx, t, store)
	})
}

func testDeleteMany(ctx context.Context, t *testing.T, st Store) {
	t.Helper()

	locations := make([]string, 0, 4)
	locations = append(locations, "delete_many/one.ssz", "delete_many/two.ssz", "delete_many/three.ssz")

	for _, location := range locations {
		if _, err := st.SaveBeaconState(ctx, &SaveParams{
			Data:     bytes.NewReader([]byte("payload")),
			Location: location,
		}); err != nil {
			t.Fatalf("Failed to save %s: %v", location, err)
		}
	}

	// One location that was never written: bulk delete is idempotent, so it must not turn the
	// batch into a failure.
	if err := st.DeleteMany(ctx, append(locations, "delete_many/never-existed.ssz")); err != nil {
		t.Fatalf("Failed to delete many: %v", err)
	}

	for _, location := range locations {
		exists, err := st.Exists(ctx, location)
		if err != nil {
			t.Fatalf("Failed to check %s: %v", location, err)
		}

		if exists {
			t.Fatalf("Expected %s to be deleted", location)
		}
	}

	if err := st.DeleteMany(ctx, nil); err != nil {
		t.Fatalf("Expected an empty delete to be a no-op, got: %v", err)
	}
}

// failingReader serves size bytes and then fails, standing in for a payload
// whose source dies part way through the upload.
type failingReader struct {
	remaining int
	err       error
}

func (r *failingReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, r.err
	}

	if len(p) > r.remaining {
		p = p[:r.remaining]
	}

	r.remaining -= len(p)

	return len(p), nil
}

// testAbortsOnAFailedRead pins the property the streaming write path depends
// on: a body that fails part way through must leave no object behind, whether
// the failure lands before the first part is sent or after several already
// have.
func testAbortsOnAFailedRead(ctx context.Context, t *testing.T, store Store) {
	t.Helper()

	readErr := errors.New("source died")

	for _, tt := range []struct {
		name string
		size int
	}{
		// Smaller than a part, so the failure arrives before anything is sent.
		{name: "SinglePart", size: 1024},
		// Larger than a part, so a multipart upload is already underway.
		{name: "Multipart", size: int(uploadPartSize) + 1024},
	} {
		t.Run(tt.name, func(t *testing.T) {
			location := "beacon_state/aborted_" + tt.name + ".ssz"

			_, err := store.SaveBeaconState(ctx, &SaveParams{
				Data:     &failingReader{remaining: tt.size, err: readErr},
				Location: location,
			})
			if err == nil {
				t.Fatal("Expected a failed read to fail the save")
			}

			exists, err := store.Exists(ctx, location)
			if err != nil {
				t.Fatalf("Failed to check existence: %v", err)
			}

			if exists {
				t.Fatal("Expected a failed read to leave no object behind")
			}
		})
	}
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
			Data:            bytes.NewReader(compressedData),
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
			Data:            bytes.NewReader(data),
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

func testExecutionPayloadEnvelope(ctx context.Context, t *testing.T, store Store) {
	t.Helper()

	location := "execution_payload_envelope/location.json"
	data := []byte(`"abc": "def"`)

	var err error

	t.Run("ExecutionPayloadEnvelope", func(t *testing.T) {
		if err = store.Healthy(ctx); err != nil {
			t.Fatalf("Store is not healthy: %v", err)
		}

		location, err = store.SaveExecutionPayloadEnvelope(ctx, &SaveParams{
			Data:            bytes.NewReader(data),
			Location:        location,
			ContentEncoding: "",
		})
		if err != nil {
			t.Fatalf("Failed to save execution payload envelope: %v", err)
		}

		retrievedData, err := store.GetExecutionPayloadEnvelope(ctx, location)
		if err != nil {
			t.Fatalf("Failed to get execution payload envelope: %v", err)
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

		if err = store.DeleteExecutionPayloadEnvelope(ctx, location); err != nil {
			t.Fatalf("Failed to delete execution payload envelope: %v", err)
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
			Data:            bytes.NewReader(data),
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
			Data:            bytes.NewReader(data),
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
			Data:            bytes.NewReader(data),
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
			Data:            bytes.NewReader(sourceData),
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
