package auth

import (
	"fmt"

	"github.com/spf13/cobra"
)

var applicationDefaultCmd = &cobra.Command{
	Use: "application-default",
}

var applicationDefaultLoginCmd = &cobra.Command{
	Use: "login",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		fmt.Print("Enter your quota project: ")
		var quotaProject string
		if _, err := fmt.Scan(&quotaProject); err != nil {
			return fmt.Errorf("unable to read quota project: %w", err)
		}

		m := DefaultApplicationCredentialManager()
		return m.AutoDetectLogin(ctx, quotaProject)
	},
}

var applicationDefaultPrintAccessTokenCmd = &cobra.Command{
	Use: "print-access-token",
	RunE: func(cmd *cobra.Command, args []string) error {
		tok, err := Token()
		if err != nil {
			return fmt.Errorf("unable to get token: %w", err)
		}
		fmt.Println(tok.AccessToken)
		return nil
	},
}
