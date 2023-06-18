package auth

import (
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:                "login",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		m := EnvApplicationCredentialManager()
		return m.AutoDetectLogin(ctx, "")
	},
}
