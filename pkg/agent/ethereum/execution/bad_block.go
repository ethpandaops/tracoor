package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type BadBlock struct {
	// The hash of the block
	Hash string `json:"hash"`
	// Block is the actual bad block
	Block json.RawMessage `json:"block"`
	// RLP is the RLP encoded block
	RLP string `json:"rlp"`
}

// BadBlockFilter decides from the hash alone whether a bad block is worth
// decoding. It runs before the block's body is materialised, which is where
// the saving is: a node repeats every bad block it has ever seen on every
// poll, and nearly all of them are already indexed.
type BadBlockFilter func(ctx context.Context, hash string) bool

// BadBlockHandler receives one decoded bad block at a time. The block is only
// valid for the duration of the call. Returning an error stops the stream.
type BadBlockHandler func(ctx context.Context, block *BadBlock) error

// errBadBlocksEnvelope reports a debug_getBadBlocks response that is not the
// JSON-RPC envelope the decoder walks.
var errBadBlocksEnvelope = errors.New("malformed debug_getBadBlocks response")

// rpcError is the error member of a JSON-RPC envelope. It satisfies rpc.Error
// so a streamed call's failure classifies the same way as one made through
// the rpc client.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var _ rpc.Error = (*rpcError)(nil)

func (e *rpcError) Error() string {
	return fmt.Sprintf("%s (code %d)", e.Message, e.Code)
}

func (e *rpcError) ErrorCode() int {
	return e.Code
}

// isEmptyJSONResult reports whether a JSON-RPC result carries nothing: an
// absent result and an explicit null both count.
func isEmptyJSONResult(data []byte) bool {
	trimmed := bytes.TrimSpace(data)

	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))
}

func (b *BadBlock) ParseBlockHeader() (*types.Header, error) {
	var header types.Header

	err := json.Unmarshal(b.Block, &header)
	if err != nil {
		return nil, err
	}

	return &header, nil
}

// decodeBadBlocks walks a debug_getBadBlocks response one token at a time, so
// the largest thing ever held in memory is a single bad block rather than the
// whole list. Nodes that never forget a bad block answer with hundreds of
// megabytes, and buffering that once per agent per poll is what used to take
// the process down.
func decodeBadBlocks(ctx context.Context, body io.Reader, want BadBlockFilter, handle BadBlockHandler) error {
	dec := json.NewDecoder(body)

	if err := expectDelim(dec, '{'); err != nil {
		return err
	}

	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return fmt.Errorf("%w: %w", errBadBlocksEnvelope, err)
		}

		key, ok := tok.(string)
		if !ok {
			return fmt.Errorf("%w: expected an object key, got %v", errBadBlocksEnvelope, tok)
		}

		switch key {
		case "result":
			if err := decodeBadBlockList(ctx, dec, want, handle); err != nil {
				return err
			}
		case "error":
			// A pointer so that an explicit "error": null reads as no error.
			var rpcErr *rpcError
			if err := dec.Decode(&rpcErr); err != nil {
				return fmt.Errorf("%w: %w", errBadBlocksEnvelope, err)
			}

			if rpcErr != nil {
				return fmt.Errorf("debug_getBadBlocks failed: %w", rpcErr)
			}
		default:
			var skip json.RawMessage
			if err := dec.Decode(&skip); err != nil {
				return fmt.Errorf("%w: %w", errBadBlocksEnvelope, err)
			}
		}
	}

	return expectDelim(dec, '}')
}

// decodeBadBlockList consumes the result array. Each element is read whole,
// but only its hash is looked at until the filter has asked for it: the body
// and RLP of a block are the expensive part, and most blocks are not wanted.
func decodeBadBlockList(ctx context.Context, dec *json.Decoder, want BadBlockFilter, handle BadBlockHandler) error {
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("%w: %w", errBadBlocksEnvelope, err)
	}

	// A node with nothing to report may answer null rather than [].
	if tok == nil {
		return nil
	}

	if delim, ok := tok.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("%w: result is not a list", errBadBlocksEnvelope)
	}

	for dec.More() {
		if err := ctx.Err(); err != nil {
			return err
		}

		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return fmt.Errorf("%w: %w", errBadBlocksEnvelope, err)
		}

		var head struct {
			Hash string `json:"hash"`
		}

		if err := json.Unmarshal(raw, &head); err != nil {
			return fmt.Errorf("%w: %w", errBadBlocksEnvelope, err)
		}

		// A bad block without a hash cannot be indexed under anything.
		if head.Hash == "" || !want(ctx, head.Hash) {
			continue
		}

		var block BadBlock
		if err := json.Unmarshal(raw, &block); err != nil {
			return fmt.Errorf("%w: %w", errBadBlocksEnvelope, err)
		}

		if err := handle(ctx, &block); err != nil {
			return err
		}
	}

	return expectDelim(dec, ']')
}

func expectDelim(dec *json.Decoder, expected json.Delim) error {
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("%w: %w", errBadBlocksEnvelope, err)
	}

	if delim, ok := tok.(json.Delim); !ok || delim != expected {
		return fmt.Errorf("%w: expected %q, got %v", errBadBlocksEnvelope, expected, tok)
	}

	return nil
}
