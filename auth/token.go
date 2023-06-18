package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/kirsle/configdir"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
)

type TokenSourceWithCacheKey interface {
	Token() (*oauth2.Token, error)
	CacheKey() string
}

type identityToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
	IdToken      string    `json:"id_token,omitempty"`
}

func tokenToIdentityToken(token *oauth2.Token) *identityToken {
	idToken, _ := token.Extra("id_token").(string)
	return &identityToken{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		IdToken:      idToken,
	}
}

func identityTokenToToken(token *identityToken, isIdentity bool) *oauth2.Token {
	accessToken := token.AccessToken
	if isIdentity {
		accessToken = token.IdToken
	}
	return &oauth2.Token{
		AccessToken:  accessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}
}

func TokenSourceCtx(ctx context.Context) (*CachingTokenSource, error) {
	return maybeGetImpersonatedTokenSource(ctx)
}

// TokenSource returns a cached application default credentials or falls back to the compute token source
func TokenSource() (*CachingTokenSource, error) {
	return TokenSourceCtx(context.Background())
}

func Token() (*oauth2.Token, error) {
	ts, err := TokenSource()
	if err != nil {
		return nil, fmt.Errorf("unable to get token: %w", err)
	}
	return ts.Token()
}

func TokenCtx(ctx context.Context) (*oauth2.Token, error) {
	ts, err := TokenSourceCtx(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get token: %w", err)
	}
	return ts.Token()
}

// easily impersonate a service account and maintain the TokenSource interface
var ImpersonateServiceAccount = ""

func maybeGetImpersonatedTokenSource(ctx context.Context) (*CachingTokenSource, error) {
	mainTs, err := getMainTokenSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get main tokensource: %w", err)
	}
	email := os.Getenv("GOOGLE_IMPERSONATE_SERVICE_ACCOUNT")
	if email == "" {
		email = ImpersonateServiceAccount
	}
	if email != "" {
		impersonateTs, err := NewGoogleImpersonateTokenSourceWrapper(ctx, email, mainTs)
		if err != nil {
			return nil, err
		}
		return NewCachingTokenSource(impersonateTs)
	}
	return mainTs, nil
}

func getMainTokenSource(ctx context.Context) (*CachingTokenSource, error) {
	var ts TokenSourceWithCacheKey
	var err error
	ts, err = EnvApplicationCredentialManager().ReadApplicationCredentials()
	if err != nil {
		ts, err = NewGoogleComputeTokenSourceWrapper(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("unable to get tokensource: %w", err)
	}
	return NewCachingTokenSource(ts)
}

func NewCachingTokenSource(ts TokenSourceWithCacheKey) (*CachingTokenSource, error) {
	cacheDir := configdir.LocalCache("gcloud-gartnera-tokens")
	err := os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("unable to ensure token cache directory: %w", err)
	}
	return &CachingTokenSource{
		ts:       ts,
		cacheDir: cacheDir,
		lock:     &sync.Mutex{},
	}, nil
}

type CachingTokenSource struct {
	ts            TokenSourceWithCacheKey
	cacheDir      string
	tok           *identityToken
	lock          *sync.Mutex
	returnIdToken bool
	cachePrefix   string
}

func (c *CachingTokenSource) cacheKey() string {
	return c.cachePrefix + c.ts.CacheKey()
}

func (c *CachingTokenSource) tokenFromDisk() (*identityToken, error) {
	cacheKey := c.cacheKey()

	cachePath := path.Join(c.cacheDir, fmt.Sprintf("%s.json", cacheKey))
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil, nil
	}

	cacheFile, err := os.Open(cachePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open cache file: %w", err)
	}
	defer cacheFile.Close()
	tok := &identityToken{}
	tokDecoder := json.NewDecoder(cacheFile)
	err = tokDecoder.Decode(tok)
	if err != nil {
		return nil, fmt.Errorf("unable to decode token: %w", err)
	}
	return tok, nil
}

func (c *CachingTokenSource) tokenToDisk(tok *identityToken) error {
	cacheKey := c.cacheKey()
	jsonCachePath := path.Join(c.cacheDir, fmt.Sprintf("%s.json", cacheKey))

	jsonTmpPattern := fmt.Sprintf("%s.json.tmp.*", cacheKey)
	jsonCacheFile, err := os.CreateTemp(c.cacheDir, jsonTmpPattern)
	if err != nil {
		return fmt.Errorf("unable to open cache file: %w", err)
	}
	defer jsonCacheFile.Close()
	tokEncoder := json.NewEncoder(jsonCacheFile)
	err = tokEncoder.Encode(tok)
	if err != nil {
		return fmt.Errorf("unable to encode token: %w", err)
	}
	jsonCacheFile.Close()
	err = os.Rename(jsonCacheFile.Name(), jsonCachePath)
	if err != nil {
		return fmt.Errorf("unable to rename tmpfile: %w", err)
	}

	// also write out the raw token for use in fallback
	rawCachePath := c.GetAccessTokenPath()
	rawTmpPattern := fmt.Sprintf("%s.tmp.*", cacheKey)
	rawCacheFile, err := os.CreateTemp(c.cacheDir, rawTmpPattern)
	if err != nil {
		return fmt.Errorf("unable to open cache file: %w", err)
	}
	if err != nil {
		return fmt.Errorf("unable to open cache file: %w", err)
	}
	defer rawCacheFile.Close()
	_, err = rawCacheFile.WriteString(tok.AccessToken)
	if err != nil {
		return fmt.Errorf("unable to write token to cache file: %w", err)
	}
	rawCacheFile.Close()
	err = os.Rename(rawCacheFile.Name(), rawCachePath)
	if err != nil {
		return fmt.Errorf("unable to rename tmpfile: %w", err)
	}
	return nil
}

func (c *CachingTokenSource) GetAccessTokenPath() string {
	cacheKey := c.cacheKey()

	return path.Join(c.cacheDir, cacheKey)
}

func (c *CachingTokenSource) Token() (*oauth2.Token, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	var err error

	if c.tok == nil {
		c.tok, err = c.tokenFromDisk()
	}
	if err != nil {
		return nil, err
	}
	if c.tok != nil && c.tok.Expiry.After(time.Now()) {
		return identityTokenToToken(c.tok, c.returnIdToken), nil
	}

	tok, err := c.ts.Token()
	if err != nil {
		return nil, fmt.Errorf("unable to get token: %w", err)
	}
	c.tok = tokenToIdentityToken(tok)
	err = c.tokenToDisk(c.tok)
	if err != nil {
		return nil, fmt.Errorf("unable to write token to disk: %w", err)
	}
	return identityTokenToToken(c.tok, c.returnIdToken), nil
}

// IdentityTokenSource clones the current token source and configured
// the clone to return the identity token in the AccessToken field.
// this only works with user accounts.
func (c *CachingTokenSource) IdentityTokenSource() *CachingTokenSource {
	return &CachingTokenSource{
		ts:            c.ts,
		cacheDir:      c.cacheDir,
		tok:           c.tok,
		lock:          c.lock,
		returnIdToken: true,
	}
}

// IdentityTokenSource gets a cached idtoken source. This can only be
// used with service accounts
func IdentityTokenSource(aud string) (*CachingTokenSource, error) {
	// get a normal token source to get the cache key
	accessTokenSource, err := TokenSource()
	if err != nil {
		return nil, fmt.Errorf("unable to get access token source: %w", err)
	}

	ts, err := idtoken.NewTokenSource(context.Background(), aud)
	if err != nil {
		return nil, fmt.Errorf("unable to get token source: %w", err)
	}
	wrapper := &tokenSourceWrapper{
		ts:       ts,
		cacheKey: fmt.Sprintf("%s-%s", aud, accessTokenSource.cacheKey()),
	}
	return NewCachingTokenSource(wrapper)
}

type tokenSourceWrapper struct {
	ts       oauth2.TokenSource
	cacheKey string
}

func (m *tokenSourceWrapper) Token() (*oauth2.Token, error) {
	return m.ts.Token()
}

func (m *tokenSourceWrapper) CacheKey() string {
	return m.cacheKey
}

func NewGoogleComputeTokenSourceWrapper(ctx context.Context) (*tokenSourceWrapper, error) {
	ts := google.ComputeTokenSource("", "https://www.googleapis.com/auth/cloud-platform")
	return &tokenSourceWrapper{
		ts:       ts,
		cacheKey: "compute-metadata",
	}, nil
}

func NewGoogleImpersonateTokenSourceWrapper(ctx context.Context, email string, parentTs oauth2.TokenSource) (*tokenSourceWrapper, error) {
	config := impersonate.CredentialsConfig{
		TargetPrincipal: email,
		Scopes:          []string{"https://www.googleapis.com/auth/cloud-platform"},
	}
	ts, err := impersonate.CredentialsTokenSource(ctx, config, option.WithTokenSource(parentTs))
	if err != nil {
		return nil, fmt.Errorf("unable to get impersonated token source: %w", err)
	}
	return &tokenSourceWrapper{
		ts:       ts,
		cacheKey: email,
	}, nil
}
