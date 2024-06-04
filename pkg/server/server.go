package server

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"

	gw "github.com/ethpandaops/tracoor/pkg/api"
	"github.com/ethpandaops/tracoor/pkg/observability"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/server/service"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/go-co-op/gocron"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	//nolint:blank-imports // Required for grpc.WithCompression
	"google.golang.org/grpc/credentials/insecure"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	log    logrus.FieldLogger
	config *Config

	services []service.GRPCService

	grpcServer    *grpc.Server
	gatewayServer *http.Server
	metricsServer *http.Server
	pprofServer   *http.Server

	db    *persistence.Indexer
	store store.Store

	grpcClientConn string
	grpcClientOpts []grpc.DialOption

	//nolint:unused // Required
	frontend http.Handler

	//nolint:unused // Required
	frontendFS fs.FS

	Started chan struct{}
}

const namespace = "tracoor_server"

func NewServer(ctx context.Context, log logrus.FieldLogger, conf *Config) (*Server, error) {
	if err := conf.Validate(); err != nil {
		return nil, err
	}

	db, err := persistence.NewIndexer("indexer", log, conf.Persistence, persistence.DefaultOptions())
	if err != nil {
		return nil, err
	}

	st, err := store.NewStore(namespace, log, conf.Store.Type, conf.Store.Config, store.DefaultOptions())
	if err != nil {
		return nil, err
	}

	var opts []grpc.DialOption

	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	services, err := service.CreateGRPCServices(ctx, log, &conf.Services, db, st, conf.Addr, opts)
	if err != nil {
		return nil, err
	}

	return &Server{
		config:         conf,
		log:            log.WithField("component", "server"),
		db:             db,
		store:          st,
		services:       services,
		grpcClientConn: conf.Addr,
		grpcClientOpts: opts,
		Started:        make(chan struct{}),
	}, nil
}

func (x *Server) Start(ctx context.Context) error {
	nctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := x.startCrons(ctx); err != nil {
		x.log.WithError(err).Fatal("Failed to start crons")
	}

	if err := x.db.Start(ctx); err != nil {
		return err
	}

	g, gCtx := errgroup.WithContext(nctx)

	g.Go(func() error {
		if err := x.startMetrics(ctx); err != nil {
			if err != http.ErrServerClosed {
				return err
			}
		}

		return nil
	})

	if x.config.PProfAddr != nil {
		g.Go(func() error {
			if err := x.startPProf(ctx); err != nil {
				if err != http.ErrServerClosed {
					return err
				}
			}

			return nil
		})
	}

	g.Go(func() error {
		if err := x.startGrpcServer(ctx); err != nil {
			return err
		}

		return nil
	})
	g.Go(func() error {
		if err := x.startGrpcGateway(ctx); err != nil {
			return err
		}

		return nil
	})
	g.Go(func() error {
		<-gCtx.Done()

		if err := x.stop(ctx); err != nil {
			return err
		}

		return nil
	})

	// Sleep for a little bit to give the http servers time to start.
	time.Sleep(2 * time.Second)

	// Signal that the server has fully started
	close(x.Started)

	err := g.Wait()

	if err != context.Canceled {
		return err
	}

	return nil
}

func (x *Server) stop(ctx context.Context) error {
	x.log.WithField("pre_stop_sleep_seconds", x.config.PreStopSleepSeconds).Info("Stopping server")

	time.Sleep(time.Duration(x.config.PreStopSleepSeconds) * time.Second)

	if x.grpcServer != nil {
		x.grpcServer.GracefulStop()
	}

	for _, s := range x.services {
		if err := s.Stop(ctx); err != nil {
			return err
		}
	}

	if err := x.db.Stop(ctx); err != nil {
		return err
	}

	if x.gatewayServer != nil {
		if err := x.gatewayServer.Shutdown(ctx); err != nil {
			return err
		}
	}

	if x.pprofServer != nil {
		if err := x.pprofServer.Shutdown(ctx); err != nil {
			return err
		}
	}

	if x.metricsServer != nil {
		if err := x.metricsServer.Shutdown(ctx); err != nil {
			return err
		}
	}

	x.log.Info("Server stopped")

	return nil
}

func (x *Server) startGrpcServer(ctx context.Context) error {
	lis, err := net.Listen("tcp", x.config.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	mb100 := 1024 * 1024 * 100

	grpc_prometheus.EnableHandlingTimeHistogram()

	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
		grpc.MaxRecvMsgSize(mb100),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     5 * time.Minute,
			MaxConnectionAge:      10 * time.Minute,
			MaxConnectionAgeGrace: 2 * time.Minute,
			Time:                  1 * time.Minute,
			Timeout:               15 * time.Second,
		}),
	}
	x.grpcServer = grpc.NewServer(opts...)

	for _, s := range x.services {
		if err := s.Start(ctx, x.grpcServer); err != nil {
			return err
		}
	}

	grpc_prometheus.Register(x.grpcServer)

	x.log.WithField("addr", x.config.Addr).Info("Starting gRPC server")

	reflection.Register(x.grpcServer)

	return x.grpcServer.Serve(lis)
}

func (x *Server) startGrpcGateway(ctx context.Context) error {
	frontend := NewFrontend(x.log)

	if err := frontend.Start(); err != nil {
		return fmt.Errorf("failed to start frontend: %v", err)
	}

	mux := runtime.NewServeMux(
		runtime.WithRoutingErrorHandler(frontend.customRoutingErrorHandler),
	)

	downloader := NewObjectDownloader(x.log, x.store, mux, x.grpcClientConn, x.grpcClientOpts)

	if err := downloader.Start(); err != nil {
		return fmt.Errorf("failed to start object downloader: %v", err)
	}

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	err := gw.RegisterAPIHandlerFromEndpoint(ctx, mux, x.config.Addr, opts)
	if err != nil {
		return err
	}

	x.gatewayServer = &http.Server{
		Addr:              x.config.GatewayAddr,
		ReadHeaderTimeout: 15 * time.Second,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
		Handler: mux,
	}

	x.log.WithField("addr", x.config.GatewayAddr).Info("Starting gRPC Gateway server")

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	return x.gatewayServer.ListenAndServe()
}

func (x *Server) startMetrics(ctx context.Context) error {
	observability.StartMetricsServer(ctx, x.config.MetricsAddr)

	return nil
}

func (x *Server) startPProf(ctx context.Context) error {
	x.log.WithField("addr", x.config.PProfAddr).Info("Starting pprof server")

	x.pprofServer = &http.Server{
		Addr:              *x.config.PProfAddr,
		ReadHeaderTimeout: 120 * time.Second,
	}

	return x.pprofServer.ListenAndServe()
}

func (x *Server) startCrons(ctx context.Context) error {
	c := gocron.NewScheduler(time.Local)

	c.StartAsync()

	return nil
}
