package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	goerrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethpandaops/beacon/pkg/beacon/api"
	"github.com/ethpandaops/tracoor/pkg/compression"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const testPayloadLocation = "beacon_state/streamed.ssz"

// payload builds a body big enough to cross the copy buffer several times, so a
// truncation lands mid-stream rather than before the first read.
func payload(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 251)
	}

	return data
}

func newTestFSStore(t *testing.T) (*store.FSStore, string) {
	t.Helper()

	base := t.TempDir()

	log := logrus.New()
	log.SetOutput(io.Discard)

	fsStore, err := store.NewFSStore("test", log, &store.FSStoreConfig{BasePath: base}, nil)
	require.NoError(t, err)

	return fsStore, base
}

// openTestResponse fetches url and wraps it the way the beacon library does, so
// the pipeline sees the same ContentLength and body semantics it does live.
func openTestResponse(t *testing.T, url string) *api.RawResponse {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)

	//nolint:bodyclose // ownership of the body passes to the pipeline, which closes it.
	rsp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return &api.RawResponse{
		Body:          rsp.Body,
		ContentType:   rsp.Header.Get("Content-Type"),
		ContentLength: rsp.ContentLength,
		Uncompressed:  rsp.Uncompressed,
		Header:        rsp.Header.Clone(),
	}
}

func requireNothingStored(t *testing.T, base string) {
	t.Helper()

	var found []string

	err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() {
			found = append(found, path)
		}

		return nil
	})
	require.NoError(t, err)
	require.Empty(t, found, "a failed stream must leave neither a published object nor a temporary file")
}

func TestStreamPayloadStoresACompleteBody(t *testing.T) {
	data := payload(512 * 1024)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))

		_, _ = w.Write(data)
	}))
	defer server.Close()

	fsStore, base := newTestFSStore(t)
	compressor := compression.NewCompressor()

	rsp := openTestResponse(t, server.URL)
	require.Equal(t, int64(len(data)), rsp.ContentLength)

	location, result, err := streamSource(
		context.Background(),
		compressor,
		sourceFromResponse(rsp),
		fsStore.SaveBeaconState,
		testPayloadLocation,
	)
	require.NoError(t, err)
	require.Equal(t, testPayloadLocation, location)

	sum := sha256.Sum256(data)
	require.Equal(t, hex.EncodeToString(sum[:]), result.ContentHash, "the hash must describe the raw bytes")
	require.Equal(t, int64(len(data)), result.RawSize)
	require.Positive(t, result.CompressedSize)
	require.Less(t, result.CompressedSize, result.RawSize, "the payload is compressible")

	stored, err := os.ReadFile(filepath.Join(base, filepath.FromSlash(testPayloadLocation)))
	require.NoError(t, err)
	require.Len(t, stored, int(result.CompressedSize))

	decompressed, err := compressor.Decompress(&stored, "streamed.ssz"+compression.Default.Extension)
	require.NoError(t, err)
	require.Equal(t, data, decompressed)
}

func TestStreamPayloadAcceptsAnUnknownContentLength(t *testing.T) {
	data := payload(256 * 1024)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Writing without a Content-Length makes the transfer chunked, which is
		// what a transparently decompressed response looks like.
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		_, _ = w.Write(data[:len(data)/2])

		flusher.Flush()

		_, _ = w.Write(data[len(data)/2:])
	}))
	defer server.Close()

	fsStore, base := newTestFSStore(t)
	compressor := compression.NewCompressor()

	rsp := openTestResponse(t, server.URL)
	require.Equal(t, int64(-1), rsp.ContentLength, "a chunked response has no length to check against")

	_, result, err := streamSource(
		context.Background(),
		compressor,
		sourceFromResponse(rsp),
		fsStore.SaveBeaconState,
		testPayloadLocation,
	)
	require.NoError(t, err)

	sum := sha256.Sum256(data)
	require.Equal(t, hex.EncodeToString(sum[:]), result.ContentHash)
	require.Equal(t, int64(len(data)), result.RawSize)

	stored, err := os.ReadFile(filepath.Join(base, filepath.FromSlash(testPayloadLocation)))
	require.NoError(t, err)

	decompressed, err := compressor.Decompress(&stored, "streamed.ssz"+compression.Default.Extension)
	require.NoError(t, err)
	require.Equal(t, data, decompressed)
}

func TestStreamPayloadStoresNothingWhenTheBodyIsTruncated(t *testing.T) {
	data := payload(512 * 1024)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Promise the whole payload, then hang up part way through it.
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write(data[:len(data)/4])

		hijacker, ok := w.(http.Hijacker)
		require.True(t, ok)

		conn, _, err := hijacker.Hijack()
		require.NoError(t, err)

		conn.Close()
	}))
	defer server.Close()

	fsStore, base := newTestFSStore(t)

	_, _, err := streamSource(
		context.Background(),
		compression.NewCompressor(),
		sourceFromResponse(openTestResponse(t, server.URL)),
		fsStore.SaveBeaconState,
		testPayloadLocation,
	)
	require.Error(t, err, "a truncated transfer must fail")

	requireNothingStored(t, base)
}

func TestStreamPayloadStoresNothingOnAShortRead(t *testing.T) {
	data := payload(64 * 1024)

	fsStore, base := newTestFSStore(t)

	// A body that ends cleanly but short of what the response promised is the
	// case a read error alone would not catch.
	rsp := &api.RawResponse{
		Body:          io.NopCloser(bytes.NewReader(data)),
		ContentLength: int64(len(data)) + 1,
	}

	_, _, err := streamSource(
		context.Background(),
		compression.NewCompressor(),
		sourceFromResponse(rsp),
		fsStore.SaveBeaconState,
		testPayloadLocation,
	)
	require.ErrorContains(t, err, "short read")

	requireNothingStored(t, base)
}

func TestStreamPayloadAbortsTheSaveWhenTheReaderFails(t *testing.T) {
	readErr := goerrors.New("connection reset")

	fsStore, base := newTestFSStore(t)

	rsp := &api.RawResponse{
		Body:          io.NopCloser(io.MultiReader(bytes.NewReader(payload(128*1024)), errReader{err: readErr})),
		ContentLength: -1,
	}

	_, _, err := streamSource(
		context.Background(),
		compression.NewCompressor(),
		sourceFromResponse(rsp),
		fsStore.SaveBeaconState,
		testPayloadLocation,
	)
	require.ErrorIs(t, err, readErr)

	requireNothingStored(t, base)
}

func TestStreamPayloadReportsAFailedSave(t *testing.T) {
	saveErr := goerrors.New("store is down")

	_, _, err := streamSource(
		context.Background(),
		compression.NewCompressor(),
		sourceFromResponse(&api.RawResponse{Body: io.NopCloser(bytes.NewReader(payload(1024))), ContentLength: -1}),
		func(context.Context, *store.SaveParams) (string, error) {
			return "", saveErr
		},
		testPayloadLocation,
	)
	require.ErrorIs(t, err, saveErr)
}

func TestStreamPayloadClosesTheResponse(t *testing.T) {
	body := &closeTrackingReader{Reader: bytes.NewReader(payload(4096))}

	_, _, err := streamSource(
		context.Background(),
		compression.NewCompressor(),
		sourceFromResponse(&api.RawResponse{Body: body, ContentLength: -1}),
		func(_ context.Context, params *store.SaveParams) (string, error) {
			_, err := io.Copy(io.Discard, params.Data)

			return params.Location, err
		},
		testPayloadLocation,
	)
	require.NoError(t, err)
	require.Equal(t, 1, body.closes)
}

type errReader struct {
	err error
}

func (r errReader) Read([]byte) (int, error) {
	return 0, r.err
}

type closeTrackingReader struct {
	io.Reader

	closes int
}

func (r *closeTrackingReader) Close() error {
	r.closes++

	return nil
}
