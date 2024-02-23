package store

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
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

func NewMockS3Store(ctx context.Context, bucket string) (Store, func() error, error) {
	minioContainer, endpoint, err := setupMinioContainer(ctx, bucket)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to setup Minio container")
	}

	deferrable := func() error {
		return minioContainer.Terminate(ctx)
	}

	store, err := NewS3Store("throwaway", logrus.New(), &S3StoreConfig{
		Endpoint:     "http://" + endpoint,
		Region:       "us-east-1",
		AccessKey:    "minioadmin",
		AccessSecret: "minioadmin",
		BucketName:   bucket,
	}, DefaultOptions().SetMetricsEnabled(false))
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create S3 store")
	}

	return store, deferrable, nil
}
