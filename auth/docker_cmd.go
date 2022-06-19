package auth

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/spf13/cobra"
)

var configureDockerCmd = &cobra.Command{
	Use:  "configure-docker",
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := os.ExpandEnv("${HOME}/.docker/config.json")
		config := configfile.New(configPath)

		configFile, err := os.Open(configPath)
		if err == nil {
			err = config.LoadFromReader(configFile)
			configFile.Close()
			if err != nil {
				return fmt.Errorf("unable to load docker config file: %w", err)
			}
		}
		for _, hostname := range args {
			config.CredentialHelpers[hostname] = "gcloud"
		}

		err = config.Save()
		if err != nil {
			return fmt.Errorf("unable to save docker config: %w", err)
		}

		return nil
	},
}

type DockerHelperRes struct {
	Secret   string
	Username string
}

var dockerHelperCmd = &cobra.Command{
	Use: "docker-helper",
	RunE: func(cmd *cobra.Command, args []string) error {
		// ignore all operations except for get
		if len(args) > 0 && args[0] != "get" {
			return nil
		}
		tok, err := Token()
		if err != nil {
			return fmt.Errorf("unable to get token: %w", err)
		}

		res := &DockerHelperRes{
			Secret:   tok.AccessToken,
			Username: "_dcgcloud_token",
		}

		encoder := json.NewEncoder(cmd.OutOrStdout())
		return encoder.Encode(res)
	},
}
