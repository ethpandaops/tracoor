//nolint:gosec // Only used in tests
package indexer

import (
	"context"
	"math/rand"

	"github.com/ethpandaops/tracoor/pkg/server/ethereum"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, length)

	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}

	return string(b)
}

func generateRandomInt64() int64 {
	return rand.Int63()
}

func NewMockIndexer(ctx context.Context, config *Config) (*Indexer, func() error, error) {
	st, cleanup, err := store.NewMockS3Store(ctx, "example-bucket")
	if err != nil {
		return &Indexer{}, nil, err
	}

	db, _, err := persistence.NewMockIndexer()
	if err != nil {
		return &Indexer{}, nil, err
	}

	index, err := NewIndexer(ctx, logrus.New(), config, db, st, &ethereum.Config{})
	if err != nil {
		if errr := cleanup(); errr != nil {
			return &Indexer{}, nil, errors.Wrapf(err, "failed to cleanup: %v", errr)
		}

		return &Indexer{}, nil, err
	}

	return index, cleanup, nil
}
