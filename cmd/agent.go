//nolint:dupl // disable duplicate code warning for cmds
package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/tracoor/pkg/agent"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	agentCfgFile string
)

// agentCmd represents the agent command
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Runs tracoor in agent mode.",
	Long: `Runs tracoor in agent mode, which means it will listen for events from
	an Ethereum beacon node and forward the data on to 	the configured sinks.`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		log.WithField("location", agentCfgFile).Info("Loading config")

		config, err := loadagentConfigFromFile(agentCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log.Info("Config loaded")

		logLevel, err := logrus.ParseLevel(config.LoggingLevel)
		if err != nil {
			log.WithField("logLevel", config.LoggingLevel).Fatal("invalid logging level")
		}

		log.SetLevel(logLevel)

		agent, err := agent.New(cmd.Context(), log, config)
		if err != nil {
			log.Fatal(err)
		}

		if err := agent.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}

		log.Info("tracoor agent exited - cya!")
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)

	agentCmd.Flags().StringVar(&agentCfgFile, "config", "agent.yaml", "config file (default is agent.yaml)")
}

func loadagentConfigFromFile(file string) (*agent.Config, error) {
	if file == "" {
		file = "agent.yaml"
	}

	config := &agent.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)

	if err != nil {
		return nil, err
	}

	type plain agent.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
