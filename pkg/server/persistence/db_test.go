package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

var (
	testDBCounter = 0
)

func newMockIndexer() (*Indexer, sqlmock.Sqlmock, error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to open mock sql db, got error: %v", err)
	}

	if db == nil {
		return nil, nil, errors.New("sql db is nil")
	}

	if mock == nil {
		return nil, nil, errors.New("mock sql is nil")
	}

	testDBCounter++

	indexer, err := NewIndexer("indexer_test", logrus.New(), Config{
		DSN:        fmt.Sprintf("file:%v?mode=memory&cache=shared", testDBCounter),
		DriverName: "sqlite",
	}, DefaultOptions().SetMetricsEnabled(false))
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create indexer, got error: %v", err)
	}

	if err := indexer.Start(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("Failed to start indexer, got error: %v", err)
	}

	return indexer, mock, nil
}
