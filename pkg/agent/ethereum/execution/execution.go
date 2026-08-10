package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/0xsequence/ethkit/ethrpc"
	"github.com/0xsequence/ethkit/ethrpc/jsonrpc"
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/execution/services"
	"github.com/sirupsen/logrus"
)

// ErrBlockNotFound is returned when the execution node does not know about
// the requested block, e.g. a gloas payload that has not been revealed yet.
var ErrBlockNotFound = errors.New("execution block not found")

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

	//nolint:errcheck // casting fine.
	return service.(*services.MetadataService)
}

func (n *Node) getDebugBlockTraceParms(ctx context.Context, client string) map[string]interface{} {
	params := map[string]interface{}{
		"disableMemory":  n.config.GetTraceDisableMemory(),
		"disableStorage": n.config.GetTraceDisableStorage(),
		"disableStack":   n.config.GetTraceDisableStack(),
	}

	// geth/reth inverts memory flag
	if client == "geth" || client == "reth" {
		params["enableMemory"] = !n.config.GetTraceDisableMemory()
		delete(params, "disableMemory")
	}

	return params
}

func (n *Node) GetRawDebugBlockTrace(ctx context.Context, hash, client string) (*[]byte, error) {
	trace, err := n.rawResult(ctx, ethrpc.NewCall(
		"debug_traceBlockByHash",
		hash,
		n.getDebugBlockTraceParms(ctx, client),
	))
	if err != nil {
		return nil, err
	}

	return &trace, nil
}

// rawResult performs call and returns its JSON-RPC result. The envelope is
// roughly as large as the result it carries, so it is confined to this frame
// and unreachable by the time the caller works with the result.
func (n *Node) rawResult(ctx context.Context, call ethrpc.Call) ([]byte, error) {
	rsp, err := n.rpc.Do(ctx, call)
	if err != nil {
		return nil, err
	}

	data := jsonrpc.Message{}
	if err := json.Unmarshal(rsp, &data); err != nil {
		return nil, err
	}

	return data.Result, nil
}

// GetBlockNumberByHash resolves an execution block number from its hash.
// Returns ErrBlockNotFound if the node does not (yet) have the block.
func (n *Node) GetBlockNumberByHash(ctx context.Context, hash string) (uint64, error) {
	data := jsonrpc.Message{}

	rsp, err := n.rpc.Do(ctx, ethrpc.NewCall(
		"eth_getBlockByHash",
		hash,
		false,
	))
	if err != nil {
		return 0, err
	}

	if err = json.Unmarshal(rsp, &data); err != nil {
		return 0, err
	}

	if len(data.Result) == 0 || string(data.Result) == "null" {
		return 0, ErrBlockNotFound
	}

	block := struct {
		Number string `json:"number"`
	}{}

	if err = json.Unmarshal([]byte(data.Result), &block); err != nil {
		return 0, err
	}

	number, err := strconv.ParseUint(strings.TrimPrefix(block.Number, "0x"), 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse block number %q: %w", block.Number, err)
	}

	return number, nil
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
