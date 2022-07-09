package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/kirsle/configdir"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

const defaultClientId = "764086051850-6qr4p6gpi6hn506pt8ejuq83di341hur.apps.googleusercontent.com"
const defaultClientSecret = "d-FL95Q19q7MQmFpd7hHD0Ty"

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
		Endpoint:     google.Endpoint,
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

func DefaultApplicationCredentialManager() *ApplicationCredentialManager {
	return &ApplicationCredentialManager{
		ClientID:     defaultClientId,
		ClientSecret: defaultClientSecret,
	}
}

func EnvApplicationCredentialManager() *ApplicationCredentialManager {
	m := DefaultApplicationCredentialManager()
	clientId, ok := os.LookupEnv("GOOGLE_APPLICATION_CLIENT_ID")
	if ok {
		m.ClientID = clientId
	}
	clientSecret, ok := os.LookupEnv("GOOGLE_APPLICATION_CLIENT_SECRET")
	if ok {
		m.ClientSecret = clientSecret
	}
	return m
}

type ApplicationCredentialManager struct {
	ClientID     string
	ClientSecret string
}

func (m *ApplicationCredentialManager) discoverAcPath() string {
	if m.ClientID != "" && m.ClientID != defaultClientId {
		credDir := configdir.LocalCache("gcloud-gartnera-credentials")
		_ = os.MkdirAll(credDir, 0755)
		return path.Join(credDir, m.ClientID+".json")
	}
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
func (m *ApplicationCredentialManager) WriteApplicationCredentials(ac *ApplicationCredentials) error {
	acPath := m.discoverAcPath()
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

func (m *ApplicationCredentialManager) ReadApplicationCredentials() (*ApplicationCredentials, error) {
	acPath := m.discoverAcPath()
	acBytes, err := ioutil.ReadFile(acPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read ac file, maybe you need to login: %w", err)
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

func (m *ApplicationCredentialManager) CodeFlowLogin(ctx context.Context, quotaProject string) error {
	conf := &oauth2.Config{
		ClientID:     m.ClientID,
		ClientSecret: m.ClientSecret,
		Scopes: []string{
			"openid",
			"https://www.googleapis.com/auth/cloud-platform",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/accounts.reauth",
		},
		Endpoint: google.Endpoint,
	}

	stateBytes := make([]byte, 25)
	_, err := rand.Read(stateBytes)
	if err != nil {
		return fmt.Errorf("unable to read state bytes: %w", err)
	}
	stateString := base64.RawStdEncoding.EncodeToString(stateBytes)

	redirectParam := oauth2.SetAuthURLParam("redirect_uri", "urn:ietf:wg:oauth:2.0:oob")
	url := conf.AuthCodeURL(stateString, oauth2.AccessTypeOffline, redirectParam)
	fmt.Fprintf(os.Stderr, "Go to the following link in your browser:\n\n%s\n\n", url)

	fmt.Fprint(os.Stderr, "Enter verification code: ")
	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return fmt.Errorf("unable to read code: %w", err)
	}
	tok, err := conf.Exchange(ctx, code, redirectParam)
	if err != nil {
		return fmt.Errorf("unable to exchange code: %w", err)
	}

	adc := &ApplicationCredentials{
		ClientID:       conf.ClientID,
		ClientSecret:   conf.ClientSecret,
		QuotaProjectId: quotaProject,
		RefreshToken:   tok.RefreshToken,
		AuthUri:        conf.Endpoint.AuthURL,
		TokenUri:       conf.Endpoint.TokenURL,
		Type:           "authorized_user",
	}
	err = m.WriteApplicationCredentials(adc)
	if err != nil {
		return fmt.Errorf("unable to save application credentials: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Login complete!")
	return nil
}

func (m *ApplicationCredentialManager) BrowserFlowLogin(ctx context.Context, quotaProject string) error {
	conf := &oauth2.Config{
		ClientID:     m.ClientID,
		ClientSecret: m.ClientSecret,
		Scopes: []string{
			"openid",
			"https://www.googleapis.com/auth/cloud-platform",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/accounts.reauth",
		},
		Endpoint: google.Endpoint,
	}

	stateBytes := make([]byte, 25)
	_, err := rand.Read(stateBytes)
	if err != nil {
		return fmt.Errorf("unable to read state bytes: %w", err)
	}
	stateString := base64.RawStdEncoding.EncodeToString(stateBytes)
	cb := &googleAccountLoginCallback{
		state:  stateString,
		values: make(chan url.Values),
	}

	redirectURL, server, err := loginServer(stateString, cb)
	if err != nil {
		return err
	}
	defer func(ctx context.Context, server *http.Server) {
		ctxCancel, cancel := context.WithCancel(ctx)
		cancel()
		_ = server.Shutdown(ctxCancel)
	}(ctx, server)

	conf.RedirectURL = redirectURL
	codeURL := conf.AuthCodeURL(stateString, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Fprintf(os.Stderr, "Opening URL to login: %s\n", codeURL)
	openURL(codeURL)

	var code string
	code, err = cb.wait()
	if err != nil {
		return fmt.Errorf("unable to login: %w", err)
	}
	var tok *oauth2.Token
	tok, err = conf.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("unable to exchange token: %w", err)
	}

	adc := &ApplicationCredentials{
		ClientID:       conf.ClientID,
		ClientSecret:   conf.ClientSecret,
		QuotaProjectId: quotaProject,
		RefreshToken:   tok.RefreshToken,
		AuthUri:        conf.Endpoint.AuthURL,
		TokenUri:       conf.Endpoint.TokenURL,
		Type:           "authorized_user",
	}
	err = m.WriteApplicationCredentials(adc)
	if err != nil {
		return fmt.Errorf("unable to save application credentials: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Login complete!")
	return nil
}

func (m *ApplicationCredentialManager) AutoDetectLogin(ctx context.Context, quotaProject string) error {
	if detectBrowserAvailable() {
		fmt.Println("browser avaliable)")
		return m.BrowserFlowLogin(ctx, quotaProject)
	}

	return m.CodeFlowLogin(ctx, quotaProject)
}

func detectBrowserAvailable() bool {
	_, isX11 := os.LookupEnv("DISPLAY")
	_, isWayland := os.LookupEnv("WAYLAND_DISPLAY")
	if isX11 || isWayland {
		return true
	}
	return false
}

func loginServer(state string, handler http.Handler) (string, *http.Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	port := strings.Split(listener.Addr().String(), ":")[1]
	redirectURL := fmt.Sprintf("http://localhost:%s", port)

	s := &http.Server{
		Handler: handler,
	}
	go s.Serve(listener) //nolint:errcheck
	return redirectURL, s, nil
}

func openURL(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		panic(fmt.Errorf("failed to open browser to login with okta: %v", err))
	}
}

type googleAccountLoginCallback struct {
	state  string
	values chan url.Values
}

func (cb *googleAccountLoginCallback) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	cb.values <- r.URL.Query()
	_, _ = w.Write([]byte("Login complete, you can close this page now"))
}

func (cb *googleAccountLoginCallback) wait() (string, error) {
	query := <-cb.values
	state := query.Get("state")
	if state != cb.state {
		return "", fmt.Errorf("invalid state in callback %s != %s", state, cb.state)
	}
	return query.Get("code"), nil
}
