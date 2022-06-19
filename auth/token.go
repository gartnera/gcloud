package auth

import (
	"context"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// TokenSource returns a cached application default credentials or falls back to the compute token source
func TokenSource() (oauth2.TokenSource, error) {
	adc, err := ReadApplicationDefaultCredentials()
	if err == nil {
		return adc, nil
	}
	return google.DefaultTokenSource(context.Background(), "https://www.googleapis.com/auth/cloud-platform")
}
