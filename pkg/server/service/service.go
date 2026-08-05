package service

import (
	"context"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/tracoor/pkg/server/ethereum"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/server/service/api"
	"github.com/ethpandaops/tracoor/pkg/server/service/indexer"
	"github.com/ethpandaops/tracoor/pkg/server/service/promotion"
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
	ServiceTypeUnknown   Type = "unknown"
	ServiceTypeIndexer   Type = indexer.ServiceType
	ServiceTypeAPI       Type = api.ServiceType
	ServiceTypePromotion Type = promotion.ServiceType
)

func CreateGRPCServices(ctx context.Context, log logrus.FieldLogger, cfg *Config, p *persistence.Indexer, c store.Store, bufferStore store.Config, grpcConn string, grpcOpts []grpc.DialOption, ethConfig *ethereum.Config) ([]GRPCService, error) {
	services := []GRPCService{}

	// Indexer
	if err := defaults.Set(&cfg.Indexer); err != nil {
		return nil, err
	}

	ind, err := indexer.NewIndexer(ctx, log, &cfg.Indexer, p, c, ethConfig)
	if err != nil {
		return nil, err
	}

	services = append(services, ind)

	// API
	if defaultErr := defaults.Set(&cfg.API); defaultErr != nil {
		return nil, defaultErr
	}

	ap, err := api.NewAPI(ctx, log, &cfg.API, c, grpcConn, grpcOpts)
	if err != nil {
		return nil, err
	}

	services = append(services, ap)

	// Promotion (opt-in)
	if err := defaults.Set(&cfg.Promotion); err != nil {
		return nil, err
	}

	if cfg.Promotion.Enabled {
		// Refuse to start when the buffer cannot outlive the processing
		// lag: the promotion service reads lagged slots from a buffer the
		// retention reaper empties on its own schedule.
		retention := cfg.Indexer.Retention.BeaconStates.Duration
		if cfg.Indexer.Retention.BeaconBlocks.Duration < retention {
			retention = cfg.Indexer.Retention.BeaconBlocks.Duration
		}

		if err := cfg.Promotion.ValidateRetention(retention); err != nil {
			return nil, err
		}

		// The corpus outlives every devnet; the buffer is reaped. They must
		// not be the same destination.
		if err := cfg.Promotion.ValidateDistinctFrom(bufferStore); err != nil {
			return nil, err
		}

		prom, err := promotion.NewPromoter(ctx, log, &cfg.Promotion, p, c)
		if err != nil {
			return nil, err
		}

		services = append(services, prom)
	}

	return services, nil
}
