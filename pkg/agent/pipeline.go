package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"

	"github.com/ethpandaops/beacon/pkg/beacon/api"
	"github.com/ethpandaops/tracoor/pkg/compression"
	"github.com/ethpandaops/tracoor/pkg/store"
)

// saveFunc is one of the store's Save* methods. Naming the shape here keeps the
// streaming pipeline from having to know which artifact it is carrying.
type saveFunc func(ctx context.Context, params *store.SaveParams) (string, error)

// streamResult describes a payload that reached the store in one piece.
type streamResult struct {
	// RawSize is the number of bytes read from the node.
	RawSize int64
	// CompressedSize is the number of bytes handed to the store.
	CompressedSize int64
	// ContentHash is the hex sha256 of the raw bytes, taken from the same pass
	// that fed the compressor, so it describes exactly what was stored.
	ContentHash string
}

// streamPayload reads rsp once, feeding a sha256 and a compressor from the same
// pass, and hands the compressed bytes straight to the store through a pipe.
// Nothing larger than the copy buffer, the compression window and the store's
// own buffering is ever held in memory.
//
// A successful open is not a successful transfer: any read error, or a short
// read against a known ContentLength, fails the whole call and propagates
// through the pipe so the store aborts rather than publishing a truncated
// object. Callers get an error and nothing to record.
//
// rsp is closed before returning, and the goroutine that reads it never
// outlives the call, so cancelling ctx aborts the transfer rather than leaving
// a reader behind.
func streamPayload(
	ctx context.Context,
	compressor *compression.Compressor,
	rsp *api.RawResponse,
	save saveFunc,
	location string,
) (string, streamResult, error) {
	if rsp == nil {
		return "", streamResult{}, fmt.Errorf("response is nil")
	}

	defer rsp.Close()

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

		rawSize, streamErr = compressInto(compressor, counter, hasher, rsp)

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

// compressInto copies rsp through the compressor and the hash in a single pass,
// returning the number of raw bytes that were read. A short read against a
// known ContentLength is reported as an error: the node closed the connection
// before it delivered what it promised, and the bytes that did arrive are not
// the artifact.
func compressInto(
	compressor *compression.Compressor,
	dst io.Writer,
	hasher hash.Hash,
	rsp *api.RawResponse,
) (int64, error) {
	writer, err := compressor.NewWriter(compression.Default, dst)
	if err != nil {
		return 0, fmt.Errorf("failed to create %s writer: %w", compression.Default.Name, err)
	}

	written, copyErr := io.Copy(io.MultiWriter(writer, hasher), rsp)

	closeErr := writer.Close()

	if copyErr != nil {
		return written, fmt.Errorf("failed to stream payload: %w", copyErr)
	}

	if closeErr != nil {
		return written, fmt.Errorf("failed to finish %s stream: %w", compression.Default.Name, closeErr)
	}

	if rsp.ContentLength >= 0 && written != rsp.ContentLength {
		return written, fmt.Errorf("short read: %d of %d bytes", written, rsp.ContentLength)
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
