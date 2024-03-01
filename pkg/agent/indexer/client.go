package indexer

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	config *Config
	log    logrus.FieldLogger

	conn *grpc.ClientConn
	pb   indexer.IndexerClient
}

func NewClient(config *Config, log logrus.FieldLogger) (*Client, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	var opts []grpc.DialOption

	if config.TLS {
		host, _, err := net.SplitHostPort(config.Address)
		if err != nil {
			return nil, fmt.Errorf("fail to get host from address: %v", err)
		}

		opts = append(opts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, host)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial(config.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("fail to dial: %v", err)
	}

	pbClient := indexer.NewIndexerClient(conn)

	return &Client{
		config: config,
		log:    log,
		conn:   conn,
		pb:     pbClient,
	}, nil
}

func (c *Client) Start(ctx context.Context) error {
	return nil
}

func (c *Client) Stop(ctx context.Context) error {
	if err := c.conn.Close(); err != nil {
		return err
	}

	return nil
}

func (c *Client) CreateBeaconState(ctx context.Context, req *indexer.CreateBeaconStateRequest) (*indexer.CreateBeaconStateResponse, error) {
	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	return c.pb.CreateBeaconState(ctx, req, grpc.UseCompressor(gzip.Name))
}

func (c *Client) ListBeaconState(ctx context.Context, req *indexer.ListBeaconStateRequest) (*indexer.ListBeaconStateResponse, error) {
	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	return c.pb.ListBeaconState(ctx, req, grpc.UseCompressor(gzip.Name))
}

func (c *Client) CreateBeaconBlock(ctx context.Context, req *indexer.CreateBeaconBlockRequest) (*indexer.CreateBeaconBlockResponse, error) {
	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	return c.pb.CreateBeaconBlock(ctx, req, grpc.UseCompressor(gzip.Name))
}

func (c *Client) ListBeaconBlock(ctx context.Context, req *indexer.ListBeaconBlockRequest) (*indexer.ListBeaconBlockResponse, error) {
	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	return c.pb.ListBeaconBlock(ctx, req, grpc.UseCompressor(gzip.Name))
}

func (c *Client) CreateBeaconBadBlock(ctx context.Context, req *indexer.CreateBeaconBadBlockRequest) (*indexer.CreateBeaconBadBlockResponse, error) {
	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	return c.pb.CreateBeaconBadBlock(ctx, req, grpc.UseCompressor(gzip.Name))
}

func (c *Client) ListBeaconBadBlock(ctx context.Context, req *indexer.ListBeaconBadBlockRequest) (*indexer.ListBeaconBadBlockResponse, error) {
	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	return c.pb.ListBeaconBadBlock(ctx, req, grpc.UseCompressor(gzip.Name))
}

func (c *Client) GetStorageHandshakeToken(ctx context.Context, req *indexer.GetStorageHandshakeTokenRequest) (*indexer.GetStorageHandshakeTokenResponse, error) {
	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	return c.pb.GetStorageHandshakeToken(ctx, req)
}

func (c *Client) CreateExecutionBlockTrace(ctx context.Context, req *indexer.CreateExecutionBlockTraceRequest) (*indexer.CreateExecutionBlockTraceResponse, error) {
	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	return c.pb.CreateExecutionBlockTrace(ctx, req, grpc.UseCompressor(gzip.Name))
}

func (c *Client) ListExecutionBlockTrace(ctx context.Context, req *indexer.ListExecutionBlockTraceRequest) (*indexer.ListExecutionBlockTraceResponse, error) {
	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	return c.pb.ListExecutionBlockTrace(ctx, req, grpc.UseCompressor(gzip.Name))
}

func (c *Client) CreateExecutionBadBlock(ctx context.Context, req *indexer.CreateExecutionBadBlockRequest) (*indexer.CreateExecutionBadBlockResponse, error) {
	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	return c.pb.CreateExecutionBadBlock(ctx, req, grpc.UseCompressor(gzip.Name))
}

func (c *Client) ListExecutionBadBlock(ctx context.Context, req *indexer.ListExecutionBadBlockRequest) (*indexer.ListExecutionBadBlockResponse, error) {
	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	return c.pb.ListExecutionBadBlock(ctx, req, grpc.UseCompressor(gzip.Name))
}
