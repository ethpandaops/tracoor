package cmd

import (
	"context"
	"os"
	"sync"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/tracoor/pkg/agent"
	"github.com/ethpandaops/tracoor/pkg/server"
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

		if err := config.Validate(); err != nil {
			log.Fatal(err)
		}

		if config.ApplyShared(); err != nil {
			log.Fatal(err)
		}

		log.Info("Single config loaded")

		logLevel, err := logrus.ParseLevel(config.Shared.LoggingLevel)
		if err != nil {
			log.WithField("logLevel", config.Shared.LoggingLevel).Fatal("invalid logging level")
		}

		log.SetLevel(logLevel)

		var wg sync.WaitGroup

		// Create a context that can be canceled
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		s, err := server.NewServer(ctx, log.WithField("container", "server"), config.Server)
		if err != nil {
			log.Fatal(err)
		}

		// Start server
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Info("Starting server")

			if err := s.Start(ctx); err != nil {
				log.Fatal(err)
			}

			log.Info("tracoor server exited - cya!")

			// Cancel the context to signal all agents to exit
			cancel()
		}()

		// Wait for the server to start before starting agents
		go func() {
			<-s.Started

			// Start all the agents
			for _, cfg := range config.Agents {
				wg.Add(1)

				go func(cfg *agent.Config) {
					defer wg.Done()

					agent, err := agent.New(ctx, log.WithField("container", "agent").WithField("name", cfg.Name), cfg)
					if err != nil {
						log.Fatal(err)
					}

					if err := agent.Start(ctx); err != nil {
						log.Fatal(err)
					}

					log.Info("tracoor agent exited - cya!")
				}(cfg)
			}
		}()

		wg.Wait()
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
