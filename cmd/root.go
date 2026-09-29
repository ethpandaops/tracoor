package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	log = logrus.New()
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "tracoor",
	Short: "",
	Long:  ``,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
//
// Signal handling belongs here, at the process entrypoint. Every command
// receives the resulting context through cmd.Context(), so a SIGTERM cancels
// one context that the whole tree shuts down from, rather than each component
// racing to react to the signal itself.
func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		stop()

		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initCommon() {
	log.SetFormatter(&logrus.TextFormatter{})

	logLevel, err := logrus.ParseLevel(logrus.InfoLevel.String())
	if err != nil {
		log.Fatal("invalid logging level")
	}

	log.SetLevel(logLevel)
}
