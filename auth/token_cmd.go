package auth

import (
	"fmt"
	"os"

	"github.com/gartnera/gcloud/helpers"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
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

func registerConfigHelperCmd(parent *cobra.Command) {
	printIdentityTokenCmd.Flags().String("audiences", os.Getenv("GCLOUD_ID_TOKEN_AUDIENCES"), "audiences for the id token")
	parent.AddCommand(printIdentityTokenCmd)
}

var printIdentityTokenCmd = &cobra.Command{
	Use:          "print-identity-token",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 || ImpersonateServiceAccount != "" {
			return helpers.ErrFallbackNoToken
		}
		audiences, _ := cmd.Flags().GetString("audiences")
		var ts oauth2.TokenSource
		var err error
		if audiences != "" {
			ts, err = IdentityTokenSource(audiences)
			if err != nil {
				return fmt.Errorf("unable to get tokensource: %w", err)
			}

		} else {
			baseTokenSource, err := TokenSource()
			if err != nil {
				return fmt.Errorf("unable to get tokensource: %w", err)
			}
			ts = baseTokenSource.IdentityTokenSource()
		}
		tok, err := ts.Token()
		if err != nil {
			return fmt.Errorf("unable to get token: %w", err)
		}
		fmt.Println(tok.AccessToken)
		return nil
	},
}
