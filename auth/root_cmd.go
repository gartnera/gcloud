package auth

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use: "auth",
}

var rootCmdInitDone = false

func GetRootCmd() *cobra.Command {
	if !rootCmdInitDone {
		rootCmd.AddCommand(applicationDefaultCmd)
		applicationDefaultCmd.AddCommand(applicationDefaultLoginCmd)
		applicationDefaultCmd.AddCommand(applicationDefaultPrintAccessTokenCmd)

		rootCmd.AddCommand(configureDockerCmd)
		rootCmd.AddCommand(dockerHelperCmd)
		rootCmd.AddCommand(loginCmd)

		rootCmd.AddCommand(printAccessTokenCmd)
		registerConfigHelperCmd(rootCmd)
		rootCmdInitDone = true
	}
	return rootCmd
}
