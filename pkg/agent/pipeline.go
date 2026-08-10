package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	goerrors "errors"
	"fmt"
	"hash"
	"io"

	"github.com/ethpandaops/beacon/pkg/beacon/api"
	"github.com/ethpandaops/tracoor/pkg/compression"
	"github.com/ethpandaops/tracoor/pkg/store"
)

// errIncompleteTransfer marks a payload that never arrived in full: the node
// closed the connection part way through, or delivered fewer bytes than it
// promised. A truncated body hashes perfectly well, so nothing derived from
// one may be stored, compared or recorded.
var errIncompleteTransfer = goerrors.New("incomplete transfer")

// saveFunc is one of the store's Save* methods. Naming the shape here keeps the
// streaming pipeline from having to know which artifact it is carrying.
type saveFunc func(ctx context.Context, params *store.SaveParams) (string, error)

// payloadSource is one readable copy of an artifact. contentLength is -1 when
// the transfer carries no length of its own, which HTTP/2, chunked encoding and
// transparent decompression all do.
type payloadSource struct {
	body          io.ReadCloser
	contentLength int64
}

// sourceFromResponse adapts an open beacon API response. The response is itself
// the reader, so closing the source closes the body.
func sourceFromResponse(rsp *api.RawResponse) *payloadSource {
	return &payloadSource{body: rsp, contentLength: rsp.ContentLength}
}

// sourceFromBytes adapts a payload that was already buffered whole, which is
// complete by construction.
func sourceFromBytes(data []byte) *payloadSource {
	return &payloadSource{body: io.NopCloser(bytes.NewReader(data)), contentLength: int64(len(data))}
}

// streamResult describes a payload that was read from a node in one piece.
type streamResult struct {
	// RawSize is the number of bytes read from the node.
	RawSize int64
	// CompressedSize is the number of bytes handed to the store. It is zero
	// when the payload was hashed without being stored.
	CompressedSize int64
	// ContentHash is the hex sha256 of the raw bytes, taken from the same pass
	// that fed the compressor, so it describes exactly what was read.
	ContentHash string
}

// streamSource reads src once, feeding a sha256 and a compressor from the same
// pass, and hands the compressed bytes straight to the store through a pipe.
// Nothing larger than the copy buffer, the compression window and the store's
// own buffering is ever held in memory.
//
// A successful open is not a successful transfer: any read error, or a short
// read against a known length, fails the whole call and propagates through the
// pipe so the store aborts rather than publishing a truncated object. Callers
// get an error and nothing to record.
//
// src is closed before returning, and the goroutine that reads it never
// outlives the call, so cancelling ctx aborts the transfer rather than leaving
// a reader behind.
func streamSource(
	ctx context.Context,
	compressor *compression.Compressor,
	src *payloadSource,
	save saveFunc,
	location string,
) (string, streamResult, error) {
	if src == nil {
		return "", streamResult{}, fmt.Errorf("payload source is nil")
	}

	defer src.body.Close()

	var (
		pipeReader, pipeWriter = io.Pipe()
		hasher                 = sha256.New()
		counter                = &countingWriter{writer: pipeWriter}
		streamErr              error
		rawSize                int64
		produced               = make(chan struct{})
	)

	go func() {
		defer close(produced)

		rawSize, streamErr = compressInto(compressor, counter, hasher, src)

		// The store is reading the other end, so this is what turns a failed
		// read into an aborted upload rather than a completed one.
		pipeWriter.CloseWithError(streamErr)
	}()

	savedLocation, saveErr := save(ctx, &store.SaveParams{
		Data:            pipeReader,
		Location:        location,
		ContentEncoding: compression.Default.ContentEncoding,
	})

	// A store that gave up early leaves the producer blocked on a write nobody
	// will read, so close the read end before waiting for it.
	pipeReader.CloseWithError(saveErr)

	<-produced

	// The transfer is judged before the upload: a store that happened to
	// succeed on a truncated body is still a failure.
	if streamErr != nil {
		return "", streamResult{}, streamErr
	}

	if saveErr != nil {
		return "", streamResult{}, saveErr
	}

	return savedLocation, streamResult{
		RawSize:        rawSize,
		CompressedSize: counter.written,
		ContentHash:    hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

// hashSource reads src once through a sha256 and discards the bytes. It is the
// path taken when the payload has already been stored by somebody else: the
// node is still read in full, because only the compress-and-upload work is
// shared, never the claim that a node served these bytes.
//
// The completeness gate is identical to the streaming path. A short read here
// would otherwise be recorded as a divergence, which is a fabricated finding.
func hashSource(src *payloadSource) (streamResult, error) {
	if src == nil {
		return streamResult{}, fmt.Errorf("payload source is nil")
	}

	defer src.body.Close()

	hasher := sha256.New()

	read, err := readPayload(src, io.Discard, hasher)
	if err != nil {
		return streamResult{}, err
	}

	return streamResult{
		RawSize:     read,
		ContentHash: hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

// compressInto copies src through the compressor and the hash in a single pass,
// returning the number of raw bytes that were read.
func compressInto(
	compressor *compression.Compressor,
	dst io.Writer,
	hasher hash.Hash,
	src *payloadSource,
) (int64, error) {
	writer, err := compressor.NewWriter(compression.Default, dst)
	if err != nil {
		return 0, fmt.Errorf("failed to create %s writer: %w", compression.Default.Name, err)
	}

	written, copyErr := readPayload(src, writer, hasher)

	closeErr := writer.Close()

	if copyErr != nil {
		return written, copyErr
	}

	if closeErr != nil {
		return written, fmt.Errorf("failed to finish %s stream: %w", compression.Default.Name, closeErr)
	}

	return written, nil
}

// readPayload copies the source into dst and the hash in one pass and reports
// whether the whole payload arrived. A short read against a known length is an
// error: the node closed the connection before it delivered what it promised,
// and the bytes that did arrive are not the artifact.
func readPayload(src *payloadSource, dst io.Writer, hasher hash.Hash) (int64, error) {
	written, err := io.Copy(io.MultiWriter(dst, hasher), src.body)
	if err != nil {
		return written, fmt.Errorf("%w: failed to stream payload: %w", errIncompleteTransfer, err)
	}

	if src.contentLength >= 0 && written != src.contentLength {
		return written, fmt.Errorf("%w: short read: %d of %d bytes", errIncompleteTransfer, written, src.contentLength)
	}

	return written, nil
}

// countingWriter records how much was written through it, which is how the
// compressed size of a stream is known without buffering it.
type countingWriter struct {
	writer  io.Writer
	written int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.writer.Write(p)
	c.written += int64(n)

	return n, err
}
