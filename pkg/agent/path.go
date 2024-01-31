package agent

import (
	"fmt"
	"path"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func CreateBeaconStateFileName(
	node string,
	network string,
	slot phase0.Slot,
	stateRoot string,
) string {
	return path.Join(
		"beacon_states",
		network,
		"slots",
		fmt.Sprintf("%d", slot),
		node,
		stateRoot,
	)
}
