package beacon

// VersionImmuneBlock is a block that is immune to version changes.
// If this structure changes, good luck with your upgrade.
type VersionImmuneBlock struct {
	Data struct {
		Message struct {
			Body struct {
				ExecutionPayload struct {
					BlockNumber string `json:"block_number"`
					BlockHash   string `json:"block_hash"`
				} `json:"execution_payload"`
			} `json:"body"`
		} `json:"message"`
	} `json:"data"`
}
