package execution

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// newRPCNode serves every JSON-RPC call from respond, echoing the request id
// the client is waiting on, and returns a Node wired to it.
func newRPCNode(t *testing.T, respond func(method string, params json.RawMessage) string) (*Node, *[]string) {
	t.Helper()

	methods := &[]string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}

		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))

		*methods = append(*methods, req.Method)

		w.Header().Set("Content-Type", "application/json")

		_, err := io.WriteString(w, `{"jsonrpc":"2.0","id":`+string(req.ID)+`,`+respond(req.Method, req.Params)+`}`)
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	client, err := rpc.DialHTTP(server.URL)
	require.NoError(t, err)

	t.Cleanup(client.Close)

	return &Node{
		config: &Config{},
		log:    logrus.New(),
		rpc:    client,
	}, methods
}

func TestGetRawDebugBlockTraceReturnsResult(t *testing.T) {
	const result = `[{"gas":21000,"failed":false,"structLogs":[]}]`

	node, methods := newRPCNode(t, func(string, json.RawMessage) string {
		return `"result":` + result
	})

	trace, err := node.GetRawDebugBlockTrace(context.Background(), "0xdeadbeef", "geth")
	require.NoError(t, err)
	require.NotNil(t, trace)

	// The envelope must be unwrapped: the caller stores the result, not the response.
	require.JSONEq(t, result, string(*trace))

	require.Equal(t, []string{"debug_traceBlockByHash"}, *methods)
}

func TestGetRawDebugBlockTraceHandsBackANullResultAsNull(t *testing.T) {
	node, _ := newRPCNode(t, func(string, json.RawMessage) string {
		return `"result":null`
	})

	trace, err := node.GetRawDebugBlockTrace(context.Background(), "0xdeadbeef", "geth")
	require.NoError(t, err)
	require.True(t, isEmptyJSONResult(*trace))
}

func TestGetRawDebugBlockTraceSurfacesTheRPCErrorCode(t *testing.T) {
	node, _ := newRPCNode(t, func(string, json.RawMessage) string {
		return `"error":{"code":-32601,"message":"the method debug_traceBlockByHash does not exist"}`
	})

	_, err := node.GetRawDebugBlockTrace(context.Background(), "0xdeadbeef", "geth")
	require.Error(t, err)

	var rpcErr rpc.Error

	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, -32601, rpcErr.ErrorCode())
}

func TestGetBlockNumberByHash(t *testing.T) {
	tests := []struct {
		name   string
		result string
		number uint64
		errIs  error
	}{
		{name: "known block", result: `{"number":"0x1a4","hash":"0xabc"}`, number: 420},
		{name: "unknown block is null", result: `null`, errIs: ErrBlockNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, _ := newRPCNode(t, func(string, json.RawMessage) string {
				return `"result":` + tt.result
			})

			number, err := node.GetBlockNumberByHash(context.Background(), "0xabc")

			if tt.errIs != nil {
				require.ErrorIs(t, err, tt.errIs)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.number, number)
		})
	}
}
