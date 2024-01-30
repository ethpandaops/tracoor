package agent

import (
	"fmt"
	"path"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func CreateBeaconStateFileName(
	prefix string,
	node string,
	network string,
	slot phase0.Slot,
	stateRoot string,
) string {
	return path.Join(
		prefix,
		"beacon_states",
		network,
		"slots",
		fmt.Sprintf("%d", slot),
		node,
		fmt.Sprintf("%s.ssz.gz", stateRoot),
	)
}
