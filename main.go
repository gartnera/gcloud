package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/gartnera/gcloud/auth"
	"github.com/gartnera/gcloud/config"
	"github.com/gartnera/gcloud/container"
	"github.com/gartnera/gcloud/helpers"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "gcloud <command> [command-flags] [command-args]",
}

func gcloudFallback() error {
	gcloudPath, err := helpers.LookPathPreamble("gcloud", "#!/bin/sh")
	if err != nil {
		return fmt.Errorf("unable to find gcloud: %w", err)
	}
	adc, err := auth.ReadApplicationDefaultCredentials()
	if err != nil {
		return fmt.Errorf("unable to read google application credentials: %w", err)
	}
	tok, err := adc.Token()
	if err != nil {
		return fmt.Errorf("unable to get access token: %w", err)
	}
	tokenFile, err := os.CreateTemp("", "gcloud-token-*")
	if err != nil {
		return fmt.Errorf("unable to create token file: %w", err)
	}
	defer os.Remove(tokenFile.Name())
	_, err = tokenFile.WriteString(tok.AccessToken)
	if err != nil {
		return fmt.Errorf("unable to write token to file: %w", err)
	}
	_ = tokenFile.Close()

	accessTokenArg := fmt.Sprintf("--access-token-file=%s", tokenFile.Name())
	args := []string{accessTokenArg}
	args = append(args, os.Args[1:]...)

	cmd := exec.Cmd{
		Path:   gcloudPath,
		Args:   args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	return cmd.Run()
}

func maybeFallback() {
	if os.Getenv("GCLOUD_NO_FALLBACK") != "" {
		return
	}
	targetCmd, _, _ := rootCmd.Find(os.Args[1:])
	if targetCmd == nil || targetCmd == rootCmd || len(targetCmd.Commands()) > 0 {
		err := gcloudFallback()
		if err != nil {
			var exerr *exec.ExitError
			if errors.As(err, &exerr) {
				os.Exit(exerr.ExitCode())
			}
			fmt.Printf("ERROR: %v", err)
		}
		os.Exit(0)
		return
	}
}

func main() {
	rootCmd.AddCommand(auth.GetRootCmd())
	rootCmd.AddCommand(config.GetRootCmd())
	rootCmd.AddCommand(container.GetRootCmd())

	// automatically fallback to google provided gcloud if we don't have a matching command
	// fallback for unknown commands, root commands, and intermediate commands (commands that have multiple children)
	maybeFallback()

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
