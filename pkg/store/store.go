package store

import (
	"context"
	"fmt"

	"github.com/ethpandaops/tracoor/pkg/yaml"
	"github.com/sirupsen/logrus"
)

// Store is an interface for different persistence implementations.
type Store interface {
	// Healthy checks if the store is healthy
	Healthy(ctx context.Context) error
	// Exists checks if the file exists in the store
	Exists(ctx context.Context, location string) (bool, error)

	// StorageHandshakeTokenExists checks if a storage handshake token exists in the store
	StorageHandshakeTokenExists(ctx context.Context, node string) (bool, error)
	// SaveStorageHandshakeToken saves a storage handshake token to the store
	SaveStorageHandshakeToken(ctx context.Context, node, data string) error
	// GetStorageHandshake fetches a storage handshake token from the store
	GetStorageHandshakeToken(ctx context.Context, node string) (string, error)

	// SaveBeaconState saves a beacon state to the store
	SaveBeaconState(ctx context.Context, data *[]byte, location string) (string, error)
	// GetBeaconState fetches a beacon state from the store
	GetBeaconState(ctx context.Context, location string) (*[]byte, error)
	// GetBeaconStateURL returns a URL for the beacon state
	GetBeaconStateURL(ctx context.Context, location string, expiry int) (string, error)
	// DeleteBeaconState deletes a beacon state from the store
	DeleteBeaconState(ctx context.Context, location string) error

	// SaveExecutionBlockTrace saves an execution block trace to the store
	SaveExecutionBlockTrace(ctx context.Context, data *[]byte, location string) (string, error)
	// GetExecutionBlockTrace fetches an execution block trace from the store
	GetExecutionBlockTrace(ctx context.Context, location string) (*[]byte, error)
	// GetExecutionBlockTraceURL returns a URL for the execution block trace
	GetExecutionBlockTraceURL(ctx context.Context, location string, expiry int) (string, error)
	// DeleteExecutionBlockTrace deletes an execution block trace from the store
	DeleteExecutionBlockTrace(ctx context.Context, location string) error

	// SaveExecutionBadBlock saves an execution bad block to the store
	SaveExecutionBadBlock(ctx context.Context, data *[]byte, location string) (string, error)
	// GetExecutionBadBlock fetches an execution bad block from the store
	GetExecutionBadBlock(ctx context.Context, location string) (*[]byte, error)
	// GetExecutionBadBlockURL returns a URL for the execution bad block
	GetExecutionBadBlockURL(ctx context.Context, location string, expiry int) (string, error)
	// DeleteExecutionBadBlock deletes an execution bad block from the store
	DeleteExecutionBadBlock(ctx context.Context, location string) error

	// PathPrefix returns the path prefix for the store
	PathPrefix() string
	// PreferURLs returns if the store prefers URLs for files
	PreferURLs() bool
}

func NewStore(namespace string, log logrus.FieldLogger, storeType Type, config yaml.RawMessage, opts *Options) (Store, error) {
	switch storeType {
	case S3StoreType:
		var s3Config *S3StoreConfig

		if err := config.Unmarshal(&s3Config); err != nil {
			return nil, err
		}

		return NewS3Store(namespace, log, s3Config, opts)
	default:
		return nil, fmt.Errorf("unknown store type: %s", storeType)
	}
}
