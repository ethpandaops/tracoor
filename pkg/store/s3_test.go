package store

import (
	"context"
	"testing"
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
}

func testBeaconState(ctx context.Context, t *testing.T, store Store) {
	t.Helper()

	location := "beacon_state/location.json"
	data := []byte(`"abc": "def"`)

	var err error

	t.Run("BeaconState", func(t *testing.T) {
		if err = store.Healthy(ctx); err != nil {
			t.Fatalf("Store is not healthy: %v", err)
		}

		location, err = store.SaveBeaconState(ctx, &data, location)
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

		location, err = store.SaveBeaconBlock(ctx, &data, location)
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

		location, err = store.SaveBeaconBadBlock(ctx, &data, location)
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

		location, err = store.SaveExecutionBlockTrace(ctx, &data, location)
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

		location, err = store.SaveExecutionBadBlock(ctx, &data, location)
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
