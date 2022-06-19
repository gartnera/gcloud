package auth

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"
)

// ApplicationCredentials is the a struct representing the application_default_credentials.json format.
// We add access_token and access_token_expiry for easy token caching and refresh.
type ApplicationCredentials struct {
	ClientID       string `json:"client_id,omitempty"`
	ClientSecret   string `json:"client_secret,omitempty"`
	QuotaProjectId string `json:"quota_project_id,omitempty"`
	RefreshToken   string `json:"refresh_token,omitempty"`
	Type           string `json:"type,omitempty"`
	AuthUri        string `json:"auth_uri,omitempty"`
	TokenUri       string `json:"token_uri,omitempty"`

	// service account fields
	ProjectID               string `json:"project_id,omitempty"`
	PrivateKeyID            string `json:"private_key_id,omitempty"`
	PrivateKey              string `json:"private_key,omitempty"`
	ClientEmail             string `json:"client_email,omitempty"`
	AuthProviderX509CertUrl string `json:"auth_provider_x509_cert_url,omitempty"`
	ClientX509CertUrl       string `json:"client_x509_cert_url,omitempty"`

	// context for token refresh operations
	ctx context.Context

	// hash of the application credentials
	hash string
}

func (a *ApplicationCredentials) userToken() (*oauth2.Token, error) {
	conf := &oauth2.Config{
		ClientID:     a.ClientID,
		ClientSecret: a.ClientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL: a.TokenUri,
		},
	}
	loadedToken := &oauth2.Token{
		RefreshToken: a.RefreshToken,
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
	return tok, nil
}

func (a *ApplicationCredentials) serviceAccountToken() (*oauth2.Token, error) {
	jwtConfig := &jwt.Config{
		Email:        a.ClientEmail,
		PrivateKey:   []byte(a.PrivateKey),
		PrivateKeyID: a.PrivateKeyID,
		Scopes:       []string{"https://www.googleapis.com/auth/cloud-platform"},
		TokenURL:     a.TokenUri,
	}
	ts := jwtConfig.TokenSource(context.Background())
	return ts.Token()
}

// Token ensures that we cache the access token.
// You should probably be using auth.Token() which conditionally calls this.
// This is slightly better than google.DefaultTokenSource because it has
// caching.
func (a *ApplicationCredentials) Token() (*oauth2.Token, error) {
	var tok *oauth2.Token
	var err error
	if a.Type == "authorized_user" {
		tok, err = a.userToken()
	} else {
		tok, err = a.serviceAccountToken()
	}
	if err != nil {
		return nil, fmt.Errorf("unable to get token: %w", err)
	}

	return tok, nil
}

func (a *ApplicationCredentials) CacheKey() string {
	if a.ClientEmail != "" {
		return a.ClientEmail
	}
	return a.hash
}

// SetContext allows you to set the context on token refresh operations
func (a *ApplicationCredentials) SetContext(ctx context.Context) {
	a.ctx = ctx
}

func discoverAcPath() string {
	acPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if acPath == "" {
		defaultConfigDir := os.Getenv("CLOUDSDK_CONFIG")
		if defaultConfigDir == "" {
			// TODO: cross platform
			defaultConfigDir = os.ExpandEnv("${HOME}/.config/gcloud")
		}
		// don't worry about this error, this is best effort anyway
		_ = os.MkdirAll(defaultConfigDir, os.ModeDir)
		acPath = path.Join(defaultConfigDir, "application_default_credentials.json")
	}
	return acPath
}

// WriteApplicationCredentials idempotently writes out the application credentials to disk
func WriteApplicationCredentials(ac *ApplicationCredentials) error {
	acPath := discoverAcPath()
	tmpPath := acPath + ".tmp"
	acFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("unable to open ac file path: %w", err)
	}
	acEncoder := json.NewEncoder(acFile)
	err = acEncoder.Encode(ac)
	if err != nil {
		return fmt.Errorf("unable to encode ac: %w", err)
	}
	acFile.Close()
	err = os.Rename(tmpPath, acPath)
	if err != nil {
		return fmt.Errorf("unable to rename tmpfile: %w", err)
	}
	return nil
}

func ReadApplicationCredentials() (*ApplicationCredentials, error) {
	acPath := discoverAcPath()
	acBytes, err := ioutil.ReadFile(acPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read ac file: %w", err)
	}
	ac := &ApplicationCredentials{}
	err = json.Unmarshal(acBytes, ac)
	if err != nil {
		return nil, fmt.Errorf("unable to decode ac: %w", err)
	}

	hashBytes := sha1.Sum(acBytes)
	ac.hash = hex.EncodeToString(hashBytes[:])
	return ac, nil
}
