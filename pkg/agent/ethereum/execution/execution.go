package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/0xsequence/ethkit/ethrpc"
	"github.com/0xsequence/ethkit/ethrpc/jsonrpc"
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/execution/services"
	"github.com/sirupsen/logrus"
)

type Node struct {
	config *Config
	log    logrus.FieldLogger
	rpc    *ethrpc.Provider

	services []services.Service

	onReadyCallbacks []func(ctx context.Context) error
}

func NewNode(log logrus.FieldLogger, conf *Config) *Node {
	return &Node{
		config:   conf,
		log:      log.WithField("module", "agent/ethereum/execution"),
		services: []services.Service{},
	}
}

func (n *Node) OnReady(_ context.Context, callback func(ctx context.Context) error) {
	n.onReadyCallbacks = append(n.onReadyCallbacks, callback)
}

func (n *Node) Start(ctx context.Context) error {
	rpc, err := ethrpc.NewProvider(n.config.NodeAddress)
	if err != nil {
		return err
	}

	metadata := services.NewMetadataService(n.log, rpc)

	svcs := []services.Service{
		&metadata,
	}

	n.rpc = rpc

	n.services = svcs

	errs := make(chan error, 1)

	go func() {
		wg := sync.WaitGroup{}

		for _, service := range n.services {
			wg.Add(1)

			service.OnReady(ctx, func(ctx context.Context) error {
				n.log.WithField("service", service.Name()).Info("Service is ready")

				wg.Done()

				return nil
			})

			n.log.WithField("service", service.Name()).Info("Starting service")

			if err := service.Start(ctx); err != nil {
				errs <- fmt.Errorf("failed to start service: %w", err)
			}

			wg.Wait()
		}

		n.log.Info("All services are ready")

		for _, callback := range n.onReadyCallbacks {
			if err := callback(ctx); err != nil {
				errs <- fmt.Errorf("failed to run on ready callback: %w", err)
			}
		}
	}()

	return nil
}

func (n *Node) Stop() error {
	return nil
}

func (n *Node) getServiceByName(name services.Name) (services.Service, error) {
	for _, service := range n.services {
		if service.Name() == name {
			return service, nil
		}
	}

	return nil, errors.New("service not found")
}

func (n *Node) Metadata() *services.MetadataService {
	service, err := n.getServiceByName("metadata")
	if err != nil {
		// This should never happen. If it does, good luck.
		return nil
	}

	return service.(*services.MetadataService)
}

func (n *Node) GetRawDebugBlockTrace(ctx context.Context, hash string) (*[]byte, error) {
	data := jsonrpc.Message{}

	rsp, err := n.rpc.Do(ctx, ethrpc.NewCall(
		"debug_traceBlockByHash",
		hash,
		map[string]interface{}{
			"disableMemory":  false,
			"disableStorage": true,
			"disableStack":   false,
		},
	))
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rsp, &data); err != nil {
		return nil, err
	}

	s := []byte(data.Result)

	return &s, nil
}

func (n *Node) GetBadBlocks(ctx context.Context) (*BadBlocksResponse, error) {
	data := jsonrpc.Message{}

	rsp, err := n.rpc.Do(ctx, ethrpc.NewCall(
		"debug_getBadBlocks",
	))
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rsp, &data); err != nil {
		return nil, err
	}

	badBlocks := []BadBlock{}
	if err := json.Unmarshal([]byte(data.Result), &badBlocks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bad blocks: %w", err)
	}

	s := BadBlocksResponse{}

	for _, block := range badBlocks {
		s[block.Hash] = block
	}

	return &s, nil
}
