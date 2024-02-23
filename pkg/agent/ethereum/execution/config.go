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

func (c *Config) GetTraceDisableMemory() bool {
	if c.TraceDisableMemory == nil {
		return false // Assuming false as the default value
	}

	return *c.TraceDisableMemory
}

func (c *Config) GetTraceDisableStack() bool {
	if c.TraceDisableStack == nil {
		return true // Assuming true as the default value
	}

	return *c.TraceDisableStack
}

func (c *Config) GetTraceDisableStorage() bool {
	if c.TraceDisableStorage == nil {
		return false // Assuming false as the default value
	}

	return *c.TraceDisableStorage
}
