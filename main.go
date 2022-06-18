package main

import (
	"os"

	"github.com/gartnera/gcloud/auth"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "gcloud <command> [command-flags] [command-args]",
}

func main() {
	pf := rootCmd.PersistentFlags()
	defaultConfigPath := os.Getenv("CLOUDSDK_CONFIG")
	if defaultConfigPath == "" {
		defaultConfigPath = os.ExpandEnv("${HOME}/.config/gcloud")
	}
	pf.String("config-path", defaultConfigPath, "path to gcloud configuration directory")

	rootCmd.AddCommand(auth.GetRootCmd())
	rootCmd.Execute()
}
