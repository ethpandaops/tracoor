package persistence

//nolint:tagliatelle // requires snake.
type Config struct {
	DSN        string `yaml:"dsn"`
	DriverName string `yaml:"driver_name"`
}

func (c *Config) Validate() error {
	return nil
}
