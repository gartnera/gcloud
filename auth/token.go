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
)

type TokenSourceWithCacheKey interface {
	Token() (*oauth2.Token, error)
	CacheKey() string
}

// TokenSource returns a cached application default credentials or falls back to the compute token source
func TokenSource() (oauth2.TokenSource, error) {
	return NewCachingTokenSource(context.Background())
}

func Token() (*oauth2.Token, error) {
	ts, err := TokenSource()
	if err != nil {
		return nil, fmt.Errorf("unable to get tokensource: %w", err)
	}
	return ts.Token()
}

func NewCachingTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	var ts TokenSourceWithCacheKey
	var err error
	ts, err = ReadApplicationCredentials()
	if err != nil {
		ts, err = NewGoogleComputeTokenSourceWrapper(context.Background())
	}
	if err != nil {
		return nil, fmt.Errorf("unable to get tokensource: %w", err)
	}
	cacheDir := configdir.LocalCache("gcloud-gartnera-tokens")
	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("unable to ensure token cache directory: %w", err)
	}
	return &CachingTokenSource{
		ts:       ts,
		cacheDir: cacheDir,
	}, nil
}

type CachingTokenSource struct {
	ts       TokenSourceWithCacheKey
	cacheDir string
	tok      *oauth2.Token
	lock     sync.RWMutex
}

func (c *CachingTokenSource) tokenFromDisk() (*oauth2.Token, error) {
	cacheKey := c.ts.CacheKey()

	cachePath := path.Join(c.cacheDir, fmt.Sprintf("%s.json", cacheKey))
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil, nil
	}

	cacheFile, err := os.Open(cachePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open cache file: %w", err)
	}
	defer cacheFile.Close()
	tok := &oauth2.Token{}
	tokDecoder := json.NewDecoder(cacheFile)
	err = tokDecoder.Decode(tok)
	if err != nil {
		return nil, fmt.Errorf("unable to decode token: %w", err)
	}
	return tok, nil
}

func (c *CachingTokenSource) tokenToDisk(tok *oauth2.Token) error {
	cacheKey := c.ts.CacheKey()

	cachePath := path.Join(c.cacheDir, fmt.Sprintf("%s.json", cacheKey))
	cachePathTmp := cachePath + ".tmp"

	cacheFile, err := os.OpenFile(cachePathTmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("unable to open cache file: %w", err)
	}
	defer cacheFile.Close()
	tokDecoder := json.NewEncoder(cacheFile)
	err = tokDecoder.Encode(tok)
	if err != nil {
		return fmt.Errorf("unable to encode token: %w", err)
	}
	cacheFile.Close()
	err = os.Rename(cachePathTmp, cachePath)
	if err != nil {
		return fmt.Errorf("unable to rename tmpfile: %w", err)
	}
	return nil
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
		return c.tok, nil
	}

	c.tok, err = c.ts.Token()
	if err != nil {
		return nil, fmt.Errorf("unable to get token: %w", err)
	}
	err = c.tokenToDisk(c.tok)
	if err != nil {
		return nil, fmt.Errorf("unable to write token to disk: %w", err)
	}
	return c.tok, nil
}

func NewGoogleComputeTokenSourceWrapper(ctx context.Context) (*GoogleComputeTokenSourceWrapper, error) {
	ts := google.ComputeTokenSource("", "https://www.googleapis.com/auth/cloud-platform")
	return &GoogleComputeTokenSourceWrapper{
		ts: ts,
	}, nil
}

type GoogleComputeTokenSourceWrapper struct {
	ts oauth2.TokenSource
}

func (m *GoogleComputeTokenSourceWrapper) Token() (*oauth2.Token, error) {
	return m.ts.Token()
}

func (m *GoogleComputeTokenSourceWrapper) CacheKey() string {
	return "compute-metadata"
}
