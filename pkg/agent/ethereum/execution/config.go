package execution

import "errors"

type Config struct {
	// The address of the Execution node to connect to
	NodeAddress         string `yaml:"nodeAddress"`
	TraceDisableMemory  *bool  `yaml:"traceDisableMemory" default:"false"`
	TraceDisableStack   *bool  `yaml:"traceDisableStack" default:"true"`
	TraceDisableStorage *bool  `yaml:"traceDisableStorage" default:"false"`
}

func (c *Config) Validate() error {
	if c.NodeAddress == "" {
		return errors.New("nodeAddress is required")
	}

	return nil
}
