package execution

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const (
	badBlockOne   = `{"hash":"0x01","block":{"number":"0x1","extraData":"0x"},"rlp":"0xf901"}`
	badBlockTwo   = `{"hash":"0x02","block":{"number":"0x2","extraData":"0x"},"rlp":"0xf902"}`
	badBlockThree = `{"hash":"0x03","block":{"number":"0x3","extraData":"0x"},"rlp":"0xf903"}`

	stallTimeout = 5 * time.Second
)

// newBadBlockNode serves handler from a test server and returns a Node that
// talks to it. Handlers that need to wait for the client to give up should
// select on done: the server only notices a disconnect once the request
// body has been drained, and it cannot be closed while a handler runs.
func newBadBlockNode(t *testing.T, handler func(w http.ResponseWriter, r *http.Request, done <-chan struct{})) *Node {
	t.Helper()

	done := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		r.Body = io.NopCloser(strings.NewReader(string(body)))

		handler(w, r, done)
	}))

	// Cleanups run last-in first-out: the handlers are released before the
	// server waits for them.
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(done) })

	log := logrus.New()
	log.SetOutput(io.Discard)

	return &Node{
		config:     &Config{NodeAddress: server.URL},
		log:        log,
		httpClient: server.Client(),
	}
}

func wantAll(context.Context, string) bool { return true }

func collect(hashes *[]string) BadBlockHandler {
	return func(_ context.Context, block *BadBlock) error {
		*hashes = append(*hashes, block.Hash)

		return nil
	}
}

func TestForEachBadBlockDecodesOnlyTheBlocksTheFilterWants(t *testing.T) {
	var request []byte

	node := newBadBlockNode(t, func(w http.ResponseWriter, r *http.Request, _ <-chan struct{}) {
		var err error

		request, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		_, err = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":[`+badBlockOne+`,`+badBlockTwo+`,`+badBlockThree+`]}`)
		require.NoError(t, err)
	})

	var (
		offered []string
		handled []string
	)

	want := func(_ context.Context, hash string) bool {
		offered = append(offered, hash)

		return hash != "0x02"
	}

	var decoded []*BadBlock

	handle := func(_ context.Context, block *BadBlock) error {
		handled = append(handled, block.Hash)
		decoded = append(decoded, block)

		return nil
	}

	require.NoError(t, node.ForEachBadBlock(context.Background(), stallTimeout, want, handle))

	require.JSONEq(t, badBlocksRequest, string(request))
	require.Equal(t, []string{"0x01", "0x02", "0x03"}, offered)
	require.Equal(t, []string{"0x01", "0x03"}, handled)

	require.Equal(t, "0xf903", decoded[1].RLP)
	require.JSONEq(t, `{"number":"0x3","extraData":"0x"}`, string(decoded[1].Block))
}

func TestForEachBadBlockHandsOverBlocksBeforeTheResponseHasEnded(t *testing.T) {
	// The server sends one block, then holds the connection open until the
	// client gives up. A buffered client would never see the block at all.
	node := newBadBlockNode(t, func(w http.ResponseWriter, r *http.Request, done <-chan struct{}) {
		_, err := io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":[`+badBlockOne+`,`)
		require.NoError(t, err)

		flusher, ok := w.(http.Flusher)
		require.True(t, ok)
		flusher.Flush()

		select {
		case <-r.Context().Done():
		case <-done:
		}
	})

	var handled []string

	err := node.ForEachBadBlock(context.Background(), 200*time.Millisecond, wantAll, collect(&handled))

	require.ErrorIs(t, err, errStalled)
	require.Equal(t, []string{"0x01"}, handled)
}

func TestForEachBadBlockKeepsGoingWhileBytesKeepArriving(t *testing.T) {
	// Each block arrives after a pause longer than the stall timeout would
	// allow in total, but shorter than the timeout itself: a slow stream is
	// not a stalled one.
	node := newBadBlockNode(t, func(w http.ResponseWriter, _ *http.Request, _ <-chan struct{}) {
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		_, err := io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":[`)
		require.NoError(t, err)
		flusher.Flush()

		for i, block := range []string{badBlockOne, badBlockTwo, badBlockThree} {
			time.Sleep(120 * time.Millisecond)

			if i > 0 {
				_, err = io.WriteString(w, ",")
				require.NoError(t, err)
			}

			_, err = io.WriteString(w, block)
			require.NoError(t, err)
			flusher.Flush()
		}

		_, err = io.WriteString(w, `]}`)
		require.NoError(t, err)
	})

	var handled []string

	require.NoError(t, node.ForEachBadBlock(context.Background(), 250*time.Millisecond, wantAll, collect(&handled)))
	require.Equal(t, []string{"0x01", "0x02", "0x03"}, handled)
}

func TestForEachBadBlockGivesUpWhenTheHeadersNeverArrive(t *testing.T) {
	node := newBadBlockNode(t, func(_ http.ResponseWriter, r *http.Request, done <-chan struct{}) {
		select {
		case <-r.Context().Done():
		case <-done:
		}
	})

	err := node.ForEachBadBlock(context.Background(), 200*time.Millisecond, wantAll, collect(new([]string)))

	require.ErrorIs(t, err, errStalled)
}

func TestForEachBadBlockStopsWhenTheHandlerFails(t *testing.T) {
	node := newBadBlockNode(t, func(w http.ResponseWriter, _ *http.Request, _ <-chan struct{}) {
		_, err := io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":[`+badBlockOne+`,`+badBlockTwo+`]}`)
		require.NoError(t, err)
	})

	errBoom := errors.New("boom")
	calls := 0

	err := node.ForEachBadBlock(context.Background(), stallTimeout, wantAll, func(context.Context, *BadBlock) error {
		calls++

		return errBoom
	})

	require.ErrorIs(t, err, errBoom)
	require.Equal(t, 1, calls)
}

func TestForEachBadBlockSurfacesTheRPCError(t *testing.T) {
	node := newBadBlockNode(t, func(w http.ResponseWriter, _ *http.Request, _ <-chan struct{}) {
		_, err := io.WriteString(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"the method debug_getBadBlocks does not exist"}}`)
		require.NoError(t, err)
	})

	err := node.ForEachBadBlock(context.Background(), stallTimeout, wantAll, collect(new([]string)))

	require.Error(t, err)
	require.Contains(t, err.Error(), "does not exist")
}

func TestForEachBadBlockRejectsAnUnexpectedStatus(t *testing.T) {
	node := newBadBlockNode(t, func(w http.ResponseWriter, _ *http.Request, _ <-chan struct{}) {
		http.Error(w, "nope", http.StatusBadGateway)
	})

	err := node.ForEachBadBlock(context.Background(), stallTimeout, wantAll, collect(new([]string)))

	require.Error(t, err)
	require.Contains(t, err.Error(), "502")
}

func TestForEachBadBlockSendsTheCredentialsInTheNodeAddress(t *testing.T) {
	var (
		user, pass string
		ok         bool
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok = r.BasicAuth()

		_, err := io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":[]}`)
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	node := &Node{
		config:     &Config{NodeAddress: strings.Replace(server.URL, "http://", "http://eth:secret@", 1)},
		log:        logrus.New(),
		httpClient: server.Client(),
	}

	require.NoError(t, node.ForEachBadBlock(context.Background(), stallTimeout, wantAll, collect(new([]string))))
	require.True(t, ok)
	require.Equal(t, "eth", user)
	require.Equal(t, "secret", pass)
}

func TestDecodeBadBlocks(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		handled []string
		errIs   error
	}{
		{
			name:    "null result is an empty list",
			body:    `{"jsonrpc":"2.0","id":1,"result":null}`,
			handled: nil,
		},
		{
			name:    "explicit null error is no error",
			body:    `{"jsonrpc":"2.0","id":1,"error":null,"result":[` + badBlockOne + `]}`,
			handled: []string{"0x01"},
		},
		{
			name:    "unknown envelope members are skipped",
			body:    `{"jsonrpc":"2.0","extra":{"deep":[1,2,{"x":null}]},"id":1,"result":[` + badBlockTwo + `]}`,
			handled: []string{"0x02"},
		},
		{
			name:    "blocks without a hash are skipped",
			body:    `{"jsonrpc":"2.0","id":1,"result":[{"block":{},"rlp":"0x"},` + badBlockThree + `]}`,
			handled: []string{"0x03"},
		},
		{
			name:  "a result that is not a list is refused",
			body:  `{"jsonrpc":"2.0","id":1,"result":{"hash":"0x01"}}`,
			errIs: errBadBlocksEnvelope,
		},
		{
			name:  "a body that is not JSON-RPC is refused",
			body:  `<html>gateway timeout</html>`,
			errIs: errBadBlocksEnvelope,
		},
		{
			name:  "a truncated list is refused",
			body:  `{"jsonrpc":"2.0","id":1,"result":[` + badBlockOne,
			errIs: errBadBlocksEnvelope,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handled []string

			err := decodeBadBlocks(context.Background(), strings.NewReader(tt.body), wantAll, collect(&handled))

			if tt.errIs != nil {
				require.ErrorIs(t, err, tt.errIs)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.handled, handled)
		})
	}
}

func TestDecodeBadBlocksStopsOnACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	body := `{"jsonrpc":"2.0","id":1,"result":[` + badBlockOne + `,` + badBlockTwo + `]}`

	var handled []string

	err := decodeBadBlocks(ctx, strings.NewReader(body), wantAll, func(_ context.Context, block *BadBlock) error {
		handled = append(handled, block.Hash)

		cancel()

		return nil
	})

	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, []string{"0x01"}, handled)
}
