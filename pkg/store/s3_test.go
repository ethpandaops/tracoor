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
		ExposedPorts: []string{"9000/tcp", "9001/tcp"},
		Env: map[string]string{
			"MINIO_ACCESS_KEY":      "minioadmin",
			"MINIO_SECRET_KEY":      "minioadmin",
			"MINIO_CONSOLE_ADDRESS": ":9001",
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

func TestS3StoreOperations(t *testing.T) {
	ctx := context.Background()
	bucket := "test"
	minioContainer, endpoint, err := setupMinioContainer(ctx, bucket)
	if err != nil {
		t.Fatalf("Failed to setup Minio container: %v", err)
	}
	defer minioContainer.Terminate(ctx)

	store, err := NewS3Store("test", logrus.New(), &S3StoreConfig{
		Endpoint:     "http://" + endpoint,
		Region:       "us-east-1",
		AccessKey:    "minioadmin",
		AccessSecret: "minioadmin",
		BucketName:   bucket,
	}, DefaultOptions())
	if err != nil {
		t.Fatalf("Failed to create S3Store: %v", err)
	}

	location := "test/location.json"
	data := []byte(`"abc": "def"`)

	t.Run("S3StoreOperations", func(t *testing.T) {
		if err := store.Healthy(ctx); err != nil {
			t.Fatalf("Store is not healthy: %v", err)
		}

		if _, err := store.SaveBeaconState(ctx, &data, location); err != nil {
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

		if err := store.DeleteBeaconState(ctx, location); err != nil {
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
