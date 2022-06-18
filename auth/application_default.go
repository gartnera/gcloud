package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"golang.org/x/oauth2"
)

// ApplicationDefaultCredentials is the a struct representing the application_default_credentials.json format.
// We add access_token and access_token_expiry for easy token caching and refresh.
type ApplicationDefaultCredentials struct {
	ClientID          string    `json:"client_id,omitempty"`
	ClientSecret      string    `json:"client_secret,omitempty"`
	QuotaProjectId    string    `json:"quota_project_id,omitempty"`
	AccessToken       string    `json:"access_token,omitempty"`
	AccessTokenExpiry time.Time `json:"access_token_expiry,omitempty"`
	RefreshToken      string    `json:"refresh_token,omitempty"`
	Type              string    `json:"type,omitempty"`
	AuthUri           string    `json:"auth_uri,omitempty"`
	TokenUri          string    `json:"token_uri,omitempty"`

	// service account fields
	ProjectID               string `json:"project_id,omitempty"`
	PrivateKeyID            string `json:"private_key_id,omitempty"`
	PrivateKey              string `json:"private_key,omitempty"`
	ClientEmail             string `json:"client_email,omitempty"`
	AuthProviderX509CertUrl string `json:"auth_provider_x509_cert_url,omitempty"`
	ClientX509CertUrl       string `json:"client_x509_cert_url,omitempty"`

	// context for token refresh operations
	ctx context.Context
}

// Token ensures that we cache the access token
func (a *ApplicationDefaultCredentials) Token() (*oauth2.Token, error) {
	conf := &oauth2.Config{
		ClientID:     a.ClientID,
		ClientSecret: a.ClientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL: a.TokenUri,
		},
	}
	loadedToken := &oauth2.Token{
		AccessToken:  a.AccessToken,
		RefreshToken: a.RefreshToken,
		Expiry:       a.AccessTokenExpiry,
		TokenType:    "refresh_token",
	}
	ctx := a.ctx
	if a.ctx == nil {
		ctx = context.Background()
	}
	tokenSource := conf.TokenSource(ctx, loadedToken)
	tok, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("unable to get/refresh token: %w", err)
	}
	if tok.AccessToken != a.AccessToken {
		a.AccessToken = tok.AccessToken
		a.AccessTokenExpiry = tok.Expiry
		// ignore this error, best effort
		_ = WriteApplicationDefaultCredentials(a)
	}
	return tok, nil
}

// SetContext allows you to set the context on token refresh operations
func (a *ApplicationDefaultCredentials) SetContext(ctx context.Context) {
	a.ctx = ctx
}

func discoverAdcPath() string {
	adcPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if adcPath == "" {
		defaultConfigDir := os.Getenv("CLOUDSDK_CONFIG")
		if defaultConfigDir == "" {
			// TODO: cross platform
			defaultConfigDir = os.ExpandEnv("${HOME}/.config/gcloud")
		}
		// don't worry about this error, this is best effort anyway
		_ = os.MkdirAll(defaultConfigDir, os.ModeDir)
		adcPath = path.Join(defaultConfigDir, "application_default_credentials.json")
	}
	return adcPath
}

// WriteApplicationDefaultCredentials idempotently writes out the application default credentials to disk
func WriteApplicationDefaultCredentials(adc *ApplicationDefaultCredentials) error {
	adcPath := discoverAdcPath()
	tmpPath := adcPath + ".tmp"
	adcFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("unable to open adc file path: %w", err)
	}
	adcEncoder := json.NewEncoder(adcFile)
	err = adcEncoder.Encode(adc)
	if err != nil {
		return fmt.Errorf("unable to encode adc: %w", err)
	}
	adcFile.Close()
	err = os.Rename(tmpPath, adcPath)
	if err != nil {
		return fmt.Errorf("unable to rename tmpfile: %w", err)
	}
	return nil
}

func ReadApplicationDefaultCredentials() (*ApplicationDefaultCredentials, error) {
	adcPath := discoverAdcPath()
	adcFile, err := os.Open(adcPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open adc file path: %w", err)
	}
	adcDecoder := json.NewDecoder(adcFile)
	adc := &ApplicationDefaultCredentials{}
	err = adcDecoder.Decode(adc)
	if err != nil {
		return nil, fmt.Errorf("unable to decode adc: %w", err)
	}
	return adc, nil
}
