package execution

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xsequence/ethkit/ethrpc"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestGetRawDebugBlockTraceReturnsResult(t *testing.T) {
	const result = `[{"gas":21000,"failed":false,"structLogs":[]}]`

	var body []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		body, err = io.ReadAll(r.Body)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")

		_, err = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":` + result + `}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	rpc, err := ethrpc.NewProvider(server.URL)
	require.NoError(t, err)

	node := &Node{
		config: &Config{},
		log:    logrus.New(),
		rpc:    rpc,
	}

	trace, err := node.GetRawDebugBlockTrace(context.Background(), "0xdeadbeef", "geth")
	require.NoError(t, err)
	require.NotNil(t, trace)

	// The envelope must be unwrapped: the caller stores the result, not the response.
	require.JSONEq(t, result, string(*trace))

	require.Contains(t, string(body), "debug_traceBlockByHash")
}
