package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/gartnera/gcloud/auth"
	"github.com/gartnera/gcloud/helpers"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "gcloud <command> [command-flags] [command-args]",
}

func gcloudFallback() error {
	gcloudPath, err := helpers.LookPathNoSelf("gcloud")
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

func main() {
	rootCmd.AddCommand(auth.GetRootCmd())

	// automatically fallback to google provided gcloud if we don't have a matching command
	targetCmd, _, _ := rootCmd.Find(os.Args[1:])
	if targetCmd == rootCmd {
		err := gcloudFallback()
		if err != nil {
			var exerr *exec.ExitError
			if errors.As(err, &exerr) {
				os.Exit(exerr.ExitCode())
			}
			fmt.Printf("ERROR: %v", err)
		}
		return
	}
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
