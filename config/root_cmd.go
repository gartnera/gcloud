package config

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use: "config",
}

var rootCmdInitDone = false

func GetRootCmd() *cobra.Command {
	if !rootCmdInitDone {
		registerConfigHelperCmd(rootCmd)
		rootCmdInitDone = true
	}
	return rootCmd
}
