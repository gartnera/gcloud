package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

type ApplicationDefaultCredentials struct {
	ClientID          string    `json:"client_id"`
	ClientSecret      string    `json:"client_secret"`
	QuotaProjectId    string    `json:"quota_project_id"`
	AccessToken       string    `json:"access_token"`
	AccessTokenExpiry time.Time `json:"access_token_expiry"`
	RefreshToken      string    `json:"refresh_token"`
	Type              string    `json:"type"`
}

var applicationDefaultCmd = &cobra.Command{
	Use: "application-default",
}

var applicationDefaultLoginCmd = &cobra.Command{
	Use: "login",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		configPath, _ := cmd.Flags().GetString("config-path")
		err := os.MkdirAll(configPath, os.ModeDir)
		if err != nil {
			return fmt.Errorf("unable to ensure config path exists: %w", err)
		}

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
		_, err = rand.Read(stateBytes)
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
			Type:              "authorized_user",
		}
		adcPath := path.Join(configPath, "application_default_credentials.json")
		adcFile, err := os.OpenFile(adcPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to open adc file path: %w", err)
		}
		adcEncoder := json.NewEncoder(adcFile)
		err = adcEncoder.Encode(adc)
		if err != nil {
			return fmt.Errorf("unable to encode adc: %w", err)
		}
		fmt.Println("Login complete!")
		return nil
	},
}
