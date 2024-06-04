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

	// SaveBeaconBlock saves a beacon block to the store
	SaveBeaconBlock(ctx context.Context, data *[]byte, location string) (string, error)
	// GetBeaconBlock fetches a beacon block from the store
	GetBeaconBlock(ctx context.Context, location string) (*[]byte, error)
	// GetBeaconBlockURL returns a URL for the beacon block
	GetBeaconBlockURL(ctx context.Context, location string, expiry int) (string, error)
	// DeleteBeaconBlock deletes a beacon block from the store
	DeleteBeaconBlock(ctx context.Context, location string) error

	// SaveBeaconBadBlock saves a beacon bad block to the store
	SaveBeaconBadBlock(ctx context.Context, data *[]byte, location string) (string, error)
	// GetBeaconBadBlock fetches a beacon bad block from the store
	GetBeaconBadBlock(ctx context.Context, location string) (*[]byte, error)
	// GetBeaconBadBlockURL returns a URL for the beacon bad block
	GetBeaconBadBlockURL(ctx context.Context, location string, expiry int) (string, error)
	// DeleteBeaconBadBlock deletes a beacon bad block from the store
	DeleteBeaconBadBlock(ctx context.Context, location string) error

	// SaveBeaconBadBlob saves a beacon bad block to the store
	SaveBeaconBadBlob(ctx context.Context, data *[]byte, location string) (string, error)
	// GetBeaconBadBlob fetches a beacon bad block from the store
	GetBeaconBadBlob(ctx context.Context, location string) (*[]byte, error)
	// GetBeaconBadBlobURL returns a URL for the beacon bad block
	GetBeaconBadBlobURL(ctx context.Context, location string, expiry int) (string, error)
	// DeleteBeaconBadBlob deletes a beacon bad block from the store
	DeleteBeaconBadBlob(ctx context.Context, location string) error

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
	// PreferURLs returns if the store prefers URLs for serving data
	PreferURLs() bool
}

func NewStore(namespace string, log logrus.FieldLogger, storeType Type, config yaml.RawMessage, opts *Options) (Store, error) {
	namespace += "_store"

	switch storeType {
	case S3StoreType:
		var s3Config *S3StoreConfig

		if err := config.Unmarshal(&s3Config); err != nil {
			return nil, err
		}

		return NewS3Store(namespace, log, s3Config, opts)
	case FSStoreType:
		var fsConfig *FSStoreConfig

		if err := config.Unmarshal(&fsConfig); err != nil {
			return nil, err
		}

		return NewFSStore(namespace, log, fsConfig, opts)
	default:
		return nil, fmt.Errorf("unknown store type: %s", storeType)
	}
}
