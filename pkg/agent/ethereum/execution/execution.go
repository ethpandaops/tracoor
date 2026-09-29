package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/execution/services"
	"github.com/sirupsen/logrus"
)

// ErrBlockNotFound is returned when the execution node does not know about
// the requested block, e.g. a gloas payload that has not been revealed yet.
var ErrBlockNotFound = errors.New("execution block not found")

type Node struct {
	config *Config
	log    logrus.FieldLogger
	rpc    *rpc.Client
	eth    *ethclient.Client

	// httpClient carries the calls whose answers are streamed rather than
	// buffered; the rpc client reads every response whole.
	httpClient *http.Client

	services []services.Service

	onReadyCallbacks []func(ctx context.Context) error
}

func NewNode(log logrus.FieldLogger, conf *Config) *Node {
	return &Node{
		config:     conf,
		log:        log.WithField("module", "agent/ethereum/execution"),
		services:   []services.Service{},
		httpClient: &http.Client{},
	}
}

func (n *Node) OnReady(_ context.Context, callback func(ctx context.Context) error) {
	n.onReadyCallbacks = append(n.onReadyCallbacks, callback)
}

func (n *Node) Start(ctx context.Context) error {
	client, err := rpc.DialContext(ctx, n.config.NodeAddress)
	if err != nil {
		return err
	}

	n.rpc = client
	n.eth = ethclient.NewClient(client)

	metadata := services.NewMetadataService(n.log, n.eth)

	svcs := []services.Service{
		&metadata,
	}

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
	if n.rpc != nil {
		n.rpc.Close()
	}

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
	trace, err := n.rawResult(ctx, "debug_traceBlockByHash", hash, n.getDebugBlockTraceParms(ctx, client))
	if err != nil {
		return nil, err
	}

	return &trace, nil
}

// rawResult performs a call and returns its JSON-RPC result verbatim. A
// response that carries no result at all reads as a JSON null, so callers
// have one shape of "nothing" to check for.
func (n *Node) rawResult(ctx context.Context, method string, args ...any) ([]byte, error) {
	var result json.RawMessage

	err := n.rpc.CallContext(ctx, &result, method, args...)
	if errors.Is(err, rpc.ErrNoResult) {
		return []byte("null"), nil
	}

	if err != nil {
		return nil, err
	}

	return result, nil
}

// BlockNumber returns the execution node's current head block number.
func (n *Node) BlockNumber(ctx context.Context) (uint64, error) {
	return n.eth.BlockNumber(ctx)
}

// GetBlockNumberByHash resolves an execution block number from its hash.
// Returns ErrBlockNotFound if the node does not (yet) have the block.
func (n *Node) GetBlockNumberByHash(ctx context.Context, hash string) (uint64, error) {
	result, err := n.rawResult(ctx, "eth_getBlockByHash", hash, false)
	if err != nil {
		return 0, err
	}

	if isEmptyJSONResult(result) {
		return 0, ErrBlockNotFound
	}

	block := struct {
		Number string `json:"number"`
	}{}

	if err = json.Unmarshal(result, &block); err != nil {
		return 0, err
	}

	number, err := strconv.ParseUint(strings.TrimPrefix(block.Number, "0x"), 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse block number %q: %w", block.Number, err)
	}

	return number, nil
}

// errStalled reports a bad block stream that stopped producing bytes.
var errStalled = errors.New("execution node stopped sending bad blocks")

// badBlocksRequest is the JSON-RPC call. debug_getBadBlocks takes no
// parameters, so it never changes.
const badBlocksRequest = `{"jsonrpc":"2.0","id":1,"method":"debug_getBadBlocks","params":[]}`

// ForEachBadBlock streams debug_getBadBlocks and hands the blocks that pass
// want to handle, one at a time, while the response is still arriving. Memory
// is bounded by one bad block rather than by the node's whole list.
//
// A stream that legitimately runs for minutes cannot live under a whole-request
// deadline, so instead the call is abandoned once no bytes have arrived for
// stall, headers included.
func (n *Node) ForEachBadBlock(ctx context.Context, stall time.Duration, want BadBlockFilter, handle BadBlockHandler) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	stalled := time.AfterFunc(stall, func() { cancel(errStalled) })
	defer stalled.Stop()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.config.NodeAddress, strings.NewReader(badBlocksRequest))
	if err != nil {
		return fmt.Errorf("failed to build debug_getBadBlocks request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("debug_getBadBlocks request failed: %w", stallCause(ctx, stall, err))
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("debug_getBadBlocks request failed: unexpected status %s", res.Status)
	}

	body := &progressReader{
		reader: res.Body,
		touch:  func() { stalled.Reset(stall) },
	}

	if err := decodeBadBlocks(ctx, body, want, handle); err != nil {
		return stallCause(ctx, stall, err)
	}

	return nil
}

// stallCause swaps the bare context error a stalled stream surfaces as for
// one that says what actually happened.
func stallCause(ctx context.Context, stall time.Duration, err error) error {
	if errors.Is(context.Cause(ctx), errStalled) {
		return fmt.Errorf("%w: no data for %s", errStalled, stall)
	}

	return err
}

// progressReader reports every byte that arrives so the stall timer can be
// pushed back.
type progressReader struct {
	reader io.Reader
	touch  func()
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.reader.Read(b)
	if n > 0 {
		p.touch()
	}

	return n, err
}
