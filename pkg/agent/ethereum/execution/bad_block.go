package execution

import (
	"encoding/json"

	"github.com/0xsequence/ethkit/go-ethereum/core/types"
)

type BadBlock struct {
	// The hash of the block
	Hash string `json:"hash"`
	// Block is the actual bad block
	Block json.RawMessage `json:"block"`
	// RLP is the RLP encoded block
	RLP string `json:"rlp"`
	// GeneratedBlockAccessList is the access list generated during block validation.
	// This may conflict with the access list in the block itself, which could indicate why the block is bad.
	// Currently only provided by Besu in debug_getBadBlocks response.
	GeneratedBlockAccessList json.RawMessage `json:"generatedBlockAccessList,omitempty"`
}

type BadBlocksResponse map[string]BadBlock

func (b *BadBlock) ParseBlockHeader() (*types.Header, error) {
	var header types.Header

	err := json.Unmarshal(b.Block, &header)
	if err != nil {
		return nil, err
	}

	return &header, nil
}
