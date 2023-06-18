package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

var autoLoginCmd = &cobra.Command{
	Use:                "autologin",
	DisableFlagParsing: true,
	SilenceUsage:       true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		tokenCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
		defer cancel()
		_, err := TokenCtx(tokenCtx)
		if err == nil {
			return nil
		}

		reqCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
		defer cancel()
		// do nothing if we're not online
		req, _ := http.NewRequestWithContext(reqCtx, "GET", "https://connectivitycheck.gstatic.com/generate_204", nil)
		_, err = http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("skipping gcloud auth autologin because of connectivity check fail: %w", err)
		}

		m := EnvApplicationCredentialManager()
		return m.AutoDetectLogin(ctx, "")
	},
}
