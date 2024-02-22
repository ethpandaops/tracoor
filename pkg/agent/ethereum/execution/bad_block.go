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
