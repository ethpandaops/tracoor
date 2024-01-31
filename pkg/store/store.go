package store

import (
	"context"
	"fmt"

	"github.com/ethpandaops/tracoor/pkg/yaml"
	"github.com/sirupsen/logrus"
)

// Store is an interface for different persistence implementations.
type Store interface {
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
	GetBeaconState(ctx context.Context, id string) (*[]byte, error)
	// Delete deletes a beacon state from the store
	DeleteState(ctx context.Context, location string) error

	// PathPrefix returns the path prefix for the store
	PathPrefix() string
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
	default:
		return nil, fmt.Errorf("unknown store type: %s", storeType)
	}
}
