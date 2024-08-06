package ethereum

type Config struct {
	Config ConfigDetails `yaml:"config"`
	Tools  ToolsConfig   `yaml:"tools"`
}

type ConfigDetails struct {
	Repository string `yaml:"repository"`
	Branch     string `yaml:"branch"`
	Path       string `yaml:"path"`
}

type ToolsConfig struct {
	Ncli CliConfig  `yaml:"ncli"`
	Lcli CliConfig  `yaml:"lcli"`
	Zcli ZcliConfig `yaml:"zcli"`
}

type CliConfig struct {
	Repository string `yaml:"repository"`
	Branch     string `yaml:"branch"`
}

type ZcliConfig struct {
	Fork string `yaml:"fork"`
}

func (c *Config) Validate() error {
	return nil
}
