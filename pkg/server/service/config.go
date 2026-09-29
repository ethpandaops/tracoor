package service

import (
	"github.com/ethpandaops/tracoor/pkg/server/service/api"
	"github.com/ethpandaops/tracoor/pkg/server/service/indexer"
	"github.com/ethpandaops/tracoor/pkg/server/service/promotion"
)

type Config struct {
	Indexer   indexer.Config   `yaml:"indexer"`
	API       api.Config       `yaml:"api"`
	Promotion promotion.Config `yaml:"promotion"`
}

func (c *Config) Validate() error {
	if err := c.Indexer.Validate(); err != nil {
		return err
	}

	if err := c.API.Validate(); err != nil {
		return err
	}

	if err := c.Promotion.Validate(); err != nil {
		return err
	}

	return nil
}
