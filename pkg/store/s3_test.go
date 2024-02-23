package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupMinioContainer(ctx context.Context, bucketName string) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ACCESS_KEY": "minioadmin",
			"MINIO_SECRET_KEY": "minioadmin",
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForListeningPort("9000/tcp").WithStartupTimeout(2 * time.Minute),
	}

	minioContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", err
	}

	endpoint, err := minioContainer.Endpoint(ctx, "")
	if err != nil {
		return nil, "", err
	}

	// Exec into the container to create the bucket
	execCmd := []string{"sh", "-c", fmt.Sprintf("mkdir -p /data/%s", bucketName)}

	_, _, execErr := minioContainer.Exec(ctx, execCmd)
	if execErr != nil {
		return nil, "", execErr
	}

	return minioContainer, endpoint, nil
}

//nolint:gocyclo // This is a test
func TestS3StoreOperations(t *testing.T) {
	ctx := context.Background()
	bucket := "mybucket"

	minioContainer, endpoint, err := setupMinioContainer(ctx, bucket)
	if err != nil {
		t.Fatalf("Failed to setup Minio container: %v", err)
	}

	defer func() {
		if err = minioContainer.Terminate(ctx); err != nil {
			t.Logf("Warning: error terminating Minio container: %v", err)
		}
	}()

	store, err := NewS3Store("throwaway", logrus.New(), &S3StoreConfig{
		Endpoint:     "http://" + endpoint,
		Region:       "us-east-1",
		AccessKey:    "minioadmin",
		AccessSecret: "minioadmin",
		BucketName:   bucket,
	}, DefaultOptions())
	if err != nil {
		t.Fatalf("Failed to create S3Store: %v", err)
	}

	location := "beacon_state/location.json"
	data := []byte(`"abc": "def"`)

	t.Run("BeaconState", func(t *testing.T) {
		if err = store.Healthy(ctx); err != nil {
			t.Fatalf("Store is not healthy: %v", err)
		}

		location, err = store.SaveBeaconState(ctx, &data, location)
		if err != nil {
			t.Fatalf("Failed to save beacon state: %v", err)
		}

		//nolint:govet // This is a test
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

	t.Run("ExecutionBlockTrace", func(t *testing.T) {
		if err = store.Healthy(ctx); err != nil {
			t.Fatalf("Store is not healthy: %v", err)
		}

		location, err = store.SaveExecutionBlockTrace(ctx, &data, location)
		if err != nil {
			t.Fatalf("Failed to save execution block trace: %v", err)
		}

		//nolint:govet // This is a test
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
