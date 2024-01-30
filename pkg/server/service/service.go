package service

import (
	"context"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/server/service/api"
	"github.com/ethpandaops/tracoor/pkg/server/service/indexer"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// GRPCService is a service that implements a single gRPC service as defined in
// our Protobuf definition.
type GRPCService interface {
	Start(ctx context.Context, server *grpc.Server) error
	Stop(ctx context.Context) error
}

type Type string

const (
	ServiceTypeUnknown Type = "unknown"
	ServiceTypeIndexer Type = indexer.ServiceType
	ServiceTypeAPI     Type = api.ServiceType
)

func CreateGRPCServices(ctx context.Context, log logrus.FieldLogger, cfg *Config, p *persistence.Indexer, c store.Store) ([]GRPCService, error) {
	services := []GRPCService{}

	// Indexer
	if err := defaults.Set(&cfg.Indexer); err != nil {
		return nil, err
	}

	ind, err := indexer.NewIndexer(ctx, log, &cfg.Indexer, p, c)
	if err != nil {
		return nil, err
	}

	services = append(services, ind)

	// API
	if err := defaults.Set(&cfg.API); err != nil {
		return nil, err
	}

	ap, err := api.NewAPI(ctx, log, &cfg.API, p, c)
	if err != nil {
		return nil, err
	}

	services = append(services, ap)

	return services, nil
}
