package persistence

const (
	KeyNode                    = "node"
	KeyBlockRoot               = "block_root"
	KeyBlockHash               = "block_hash"
	KeyBlockNumber             = "block_number"
	KeyExecutionImplementation = "execution_implementation"
	KeyNetwork                 = "network"
	KeySlot                    = "slot"
	KeyEpoch                   = "epoch"
	KeyStateRoot               = "state_root"
	KeyContentEncoding         = "content_encoding"
	KeyNodeVersion             = "node_version"
	KeyLocation                = "location"
	KeyFetchedAt               = "fetched_at"
	KeyBeaconImplementation    = "beacon_implementation"
)

//nolint:tagliatelle // requires snake.
type Config struct {
	DSN        string `yaml:"dsn"`
	DriverName string `yaml:"driver_name"`
}

func (c *Config) Validate() error {
	return nil
}
