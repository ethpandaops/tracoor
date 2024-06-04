package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/tracoor/pkg/single"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	singleCfgFile string
)

// singleCmd represents the single command
var singleCmd = &cobra.Command{
	Use:   "single",
	Short: "Runs tracoor in Single mode, with one server and multiple agents.",
	Long: `Runs tracoor in Single mode, which means it will start a server and multiple agents,
	listening to gRPC requests from tracoor agent nodes and forwarding the data on to the configured sinks.`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		config, err := loadSingleConfigFromFile(singleCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		err = config.Validate()
		if err != nil {
			log.Fatal(err)
		}

		err = config.ApplyShared()
		if err != nil {
			log.Fatal(err)
		}

		log.Info("Single config loaded")

		logLevel, err := logrus.ParseLevel(config.Shared.LoggingLevel)
		if err != nil {
			log.WithField("logLevel", config.Shared.LoggingLevel).Fatal("invalid logging level")
		}

		log.SetLevel(logLevel)

		s := single.New(log, config)

		if err := s.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(singleCmd)

	singleCmd.Flags().StringVar(&singleCfgFile, "single-config", "single.yaml", "single config file (default is single.yaml)")
}

func loadSingleConfigFromFile(file string) (*single.Config, error) {
	if file == "" {
		file = "single.yaml"
	}

	config := &single.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)

	if err != nil {
		return nil, err
	}

	type plain single.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
