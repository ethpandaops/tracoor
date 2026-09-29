package beacon

// VersionImmuneBlock is a block that is immune to version changes.
// If this structure changes, good luck with your upgrade.
//
// Pre-gloas blocks embed the execution payload in the body. Gloas (ePBS)
// blocks instead carry a signed execution payload bid, which commits to the
// payload's block hash but not its block number.
//
//nolint:tagliatelle // requires snake.
type VersionImmuneBlock struct {
	Data struct {
		Message struct {
			Body struct {
				ExecutionPayload struct {
					BlockNumber string `json:"block_number"`
					BlockHash   string `json:"block_hash"`
				} `json:"execution_payload"`
				SignedExecutionPayloadBid struct {
					Message struct {
						BlockHash string `json:"block_hash"`
					} `json:"message"`
				} `json:"signed_execution_payload_bid"`
			} `json:"body"`
		} `json:"message"`
	} `json:"data"`
}
