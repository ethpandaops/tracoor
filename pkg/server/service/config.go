package service

import (
	"github.com/ethpandaops/tracoor/pkg/server/service/api"
	"github.com/ethpandaops/tracoor/pkg/server/service/indexer"
)

type Config struct {
	Indexer indexer.Config `yaml:"indexer"`
	API     api.Config     `yaml:"api"`
}

func (c *Config) Validate() error {
	if err := c.Indexer.Validate(); err != nil {
		return err
	}

	if err := c.API.Validate(); err != nil {
		return err
	}

	return nil
}
