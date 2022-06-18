package auth

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// TokenSource returns a cached application default credentials or falls back to the compute token source
func TokenSource() oauth2.TokenSource {
	adc, err := ReadApplicationDefaultCredentials()
	if err == nil {
		return adc
	}
	return google.ComputeTokenSource("")
}
