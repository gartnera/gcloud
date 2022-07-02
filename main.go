package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gartnera/gcloud/auth"
	"github.com/gartnera/gcloud/config"
	"github.com/gartnera/gcloud/container"
	"github.com/gartnera/gcloud/helpers"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "gcloud <command> [command-flags] [command-args]",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		auth.ImpersonateServiceAccount, _ = cmd.Flags().GetString("impersonate-service-account")
	},
}

func stripImpersonate(args []string) []string {
	var res []string
	for _, arg := range args {
		if strings.Contains(arg, "impersonate-service-account") {
			continue
		}
		res = append(res, arg)
	}
	return res
}

func gcloudFallback(shareToken bool) error {
	gcloudPath, err := helpers.LookPathPreamble("gcloud", "#!/bin/sh")
	if err != nil {
		return fmt.Errorf("unable to find gcloud: %w", err)
	}
	args := os.Args[1:]
	if shareToken {
		args = stripImpersonate(args)
		ts, err := auth.TokenSource()
		if err != nil {
			return fmt.Errorf("unable to get tokensource: %w", err)
		}
		// ensure we have a valid token
		_, err = ts.Token()
		if err != nil {
			return fmt.Errorf("unable to get access token: %w", err)
		}

		accessTokenArg := fmt.Sprintf("--access-token-file=%s", ts.GetAccessTokenPath())
		accessTokenArgs := []string{accessTokenArg}
		args = append(accessTokenArgs, args...)
	}
	args = append([]string{os.Args[0]}, args...)

	cmd := exec.Cmd{
		Path:   gcloudPath,
		Args:   args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	return cmd.Run()
}

func gcloudFallbackExit(useToken bool) {
	err := gcloudFallback(useToken)
	if err != nil {
		var exerr *exec.ExitError
		if errors.As(err, &exerr) {
			os.Exit(exerr.ExitCode())
		}
		fmt.Printf("Error: %v\n", err)
	}
	os.Exit(0)
}

func maybeFallback() {
	if os.Getenv("GCLOUD_NO_FALLBACK") != "" {
		return
	}
	targetCmd, _, _ := rootCmd.Find(os.Args[1:])
	if targetCmd == nil || targetCmd == rootCmd || len(targetCmd.Commands()) > 0 {
		gcloudFallbackExit(true)
	}
}

func main() {
	rootCmd.PersistentFlags().String("impersonate-service-account", "", "service account email to impersonate")
	rootCmd.AddCommand(auth.GetRootCmd())
	rootCmd.AddCommand(config.GetRootCmd())
	rootCmd.AddCommand(container.GetRootCmd())

	// automatically fallback to google provided gcloud if we don't have a matching command
	// fallback for unknown commands, root commands, and intermediate commands (commands that have multiple children)
	maybeFallback()

	err := rootCmd.Execute()
	if err != nil {
		// we matched commands, but don' implement what the user has requested
		if err == helpers.ErrFallback {
			gcloudFallbackExit(true)
		}
		if err == helpers.ErrFallbackNoToken {
			gcloudFallbackExit(false)
		}
		os.Exit(1)
	}
}
