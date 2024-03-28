package beacon

import "github.com/pkg/errors"

type Config struct {
	// The address of the Beacon node to connect to
	NodeAddress string `yaml:"nodeAddress"`
	// BeaconNodeHeaders is a map of headers to send to the beacon node.
	NodeHeaders map[string]string `yaml:"nodeHeaders"`
	// BeaconSubscriptions is a list of beacon subscriptions to subscribe to.
	BeaconSubscriptions *[]string `yaml:"beaconSubscriptions"`
	// InvalidGossipVerifiedBlocksPath is the path to watch for invalid gossip verified blocks from the beacon node.
	InvalidGossipVerifiedBlocksPath *string `yaml:"invalidGossipVerifiedBlocksPath"`
	// InvalidGossipVerifiedBlobsPath is the path to watch for invalid gossip verified blobs from the beacon node.
	InvalidGossipVerifiedBlobsPath *string `yaml:"invalidGossipVerifiedBlobsPath"`
}

func (c *Config) Validate() error {
	if c.NodeAddress == "" {
		return errors.New("beaconNodeAddress is required")
	}

	return nil
}
