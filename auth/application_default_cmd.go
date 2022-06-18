package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
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

		conf := &oauth2.Config{
			ClientID:     "764086051850-6qr4p6gpi6hn506pt8ejuq83di341hur.apps.googleusercontent.com",
			ClientSecret: "d-FL95Q19q7MQmFpd7hHD0Ty",
			Scopes: []string{
				"openid",
				"https://www.googleapis.com/auth/cloud-platform",
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/accounts.reauth",
			},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/auth",
				TokenURL: "https://oauth2.googleapis.com/token",
			},
		}

		stateBytes := make([]byte, 25)
		_, err := rand.Read(stateBytes)
		if err != nil {
			return fmt.Errorf("unable to read state bytes: %w", err)
		}
		stateString := base64.RawStdEncoding.EncodeToString(stateBytes)

		redirectParam := oauth2.SetAuthURLParam("redirect_uri", "urn:ietf:wg:oauth:2.0:oob")
		url := conf.AuthCodeURL(stateString, oauth2.AccessTypeOffline, redirectParam)
		fmt.Printf("Go to the following link in your browser:\n\n%s\n\n", url)

		fmt.Print("Enter verification code: ")
		var code string
		if _, err := fmt.Scan(&code); err != nil {
			return fmt.Errorf("unable to read code: %w", err)
		}
		tok, err := conf.Exchange(ctx, code, redirectParam)
		if err != nil {
			return fmt.Errorf("unable to exchange code: %w", err)
		}

		adc := &ApplicationDefaultCredentials{
			ClientID:          conf.ClientID,
			ClientSecret:      conf.ClientSecret,
			QuotaProjectId:    quotaProject,
			AccessToken:       tok.AccessToken,
			AccessTokenExpiry: tok.Expiry.UTC(),
			RefreshToken:      tok.RefreshToken,
			AuthUri:           conf.Endpoint.AuthURL,
			TokenUri:          conf.Endpoint.TokenURL,
			Type:              "authorized_user",
		}
		err = WriteApplicationDefaultCredentials(adc)
		if err != nil {
			return fmt.Errorf("unable to save application default credentials: %w", err)
		}
		fmt.Println("Login complete!")
		return nil
	},
}

var applicationDefaultPrintAccessTokenCmd = &cobra.Command{
	Use: "print-access-token",
	RunE: func(cmd *cobra.Command, args []string) error {
		adc, err := ReadApplicationDefaultCredentials()
		if err != nil {
			return fmt.Errorf("unable to load application default credentials: %w", err)
		}
		tok, err := adc.Token()
		if err != nil {
			return fmt.Errorf("unable to get token: %w", err)
		}
		fmt.Println(tok.AccessToken)
		return nil
	},
}
