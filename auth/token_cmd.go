package auth

import (
	"fmt"

	"github.com/spf13/cobra"
)

var printAccessTokenCmd = &cobra.Command{
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

var printIdentityTokenCmd = &cobra.Command{
	Use: "print-identity-token",
	RunE: func(cmd *cobra.Command, args []string) error {
		ts, err := TokenSource()
		if err != nil {
			return fmt.Errorf("unable to get tokensource: %w", err)
		}
		ts = ts.IdentityTokenSource()
		tok, err := ts.Token()
		if err != nil {
			return fmt.Errorf("unable to get token: %w", err)
		}
		fmt.Println(tok.AccessToken)
		return nil
	},
}
