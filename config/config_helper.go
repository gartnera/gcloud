package config

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gartnera/gcloud/auth"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ConfigHelperOutputCredential struct {
	AccessToken string    `json:"access_token" yaml:"access_token"`
	IDToken     string    `json:"id_token" yaml:"id_token"`
	TokenExpiry time.Time `json:"token_expiry" yaml:"token_expiry"`
}

type ConfigHelperOutput struct {
	Credential *ConfigHelperOutputCredential `json:"credential" yaml:"credential"`
}

type ExecCredential struct {
	APIVersion string               `json:"apiVersion"`
	Kind       string               `json:"kind"`
	Status     ExecCredentialStatus `json:"status"`
}

type ExecCredentialStatus struct {
	ExpirationTimestamp time.Time `json:"expirationTimestamp"`
	Token               string    `json:"token"`
}

var configHelperCmd = &cobra.Command{
	Use: "config-helper",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFormat, _ := cmd.Flags().GetString("format")
		ts, err := auth.TokenSource()
		if err != nil {
			return fmt.Errorf("unable to get tokensource: %w", err)
		}
		token, err := ts.Token()
		if err != nil {
			return fmt.Errorf("unable to get token: %w", err)
		}
		output := &ConfigHelperOutput{
			Credential: &ConfigHelperOutputCredential{
				AccessToken: token.AccessToken,
				TokenExpiry: token.Expiry,
			},
		}
		jsonEncoder := json.NewEncoder(cmd.OutOrStdout())
		if outputFormat == "json" {
			err = jsonEncoder.Encode(output)
		} else if outputFormat == "json(credential)" {
			err = jsonEncoder.Encode(output.Credential)
		} else if outputFormat == "yaml" {
			encoder := yaml.NewEncoder(cmd.OutOrStdout())
			err = encoder.Encode(output)
		} else if outputFormat == "client.authentication.k8s.io/v1beta1" {
			outputv1 := &ExecCredential{
				APIVersion: "client.authentication.k8s.io/v1beta1",
				Kind:       "ExecCredential",
				Status: ExecCredentialStatus{
					Token:               token.AccessToken,
					ExpirationTimestamp: token.Expiry,
				}}
			err = jsonEncoder.Encode(outputv1)
		} else {
			return fmt.Errorf("invalid output format: %s", outputFormat)
		}

		return err
	},
}

func registerConfigHelperCmd(parent *cobra.Command) {
	configHelperCmd.Flags().StringP("format", "o", "yaml", "output format")
	parent.AddCommand(configHelperCmd)
}
