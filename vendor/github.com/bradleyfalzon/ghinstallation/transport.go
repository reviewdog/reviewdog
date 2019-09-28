package ghinstallation

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

const (
	// acceptHeader is the GitHub Integrations Preview Accept header.
	acceptHeader = "application/vnd.github.machine-man-preview+json"
	apiBaseURL   = "https://api.github.com"
)

// Transport provides a http.RoundTripper by wrapping an existing
// http.RoundTripper and provides GitHub Apps authentication as an
// installation.
//
// Client can also be overwritten, and is useful to change to one which
// provides retry logic if you do experience retryable errors.
//
// See https://developer.github.com/apps/building-integrations/setting-up-and-registering-github-apps/about-authentication-options-for-github-apps/
type Transport struct {
	BaseURL        string            // BaseURL is the scheme and host for GitHub API, defaults to https://api.github.com
	Client         Client            // Client to use to refresh tokens, defaults to http.Client with provided transport
	tr             http.RoundTripper // tr is the underlying roundtripper being wrapped
	integrationID  int               // integrationID is the GitHub Integration's Installation ID
	installationID int               // installationID is the GitHub Integration's Installation ID
	appsTransport  *AppsTransport

	mu    *sync.Mutex  // mu protects token
	token *accessToken // token is the installation's access token
}

// accessToken is an installation access token response from GitHub
type accessToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

var _ http.RoundTripper = &Transport{}

// NewKeyFromFile returns a Transport using a private key from file.
func NewKeyFromFile(tr http.RoundTripper, integrationID, installationID int, privateKeyFile string) (*Transport, error) {
	privateKey, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("could not read private key: %s", err)
	}
	return New(tr, integrationID, installationID, privateKey)
}

// Client is a HTTP client which sends a http.Request and returns a http.Response
// or an error.
type Client interface {
	Do(*http.Request) (*http.Response, error)
}

// New returns an Transport using private key. The key is parsed
// and if any errors occur the error is non-nil.
//
// The provided tr http.RoundTripper should be shared between multiple
// installations to ensure reuse of underlying TCP connections.
//
// The returned Transport's RoundTrip method is safe to be used concurrently.
func New(tr http.RoundTripper, integrationID, installationID int, privateKey []byte) (*Transport, error) {
	t := &Transport{
		tr:             tr,
		integrationID:  integrationID,
		installationID: installationID,
		BaseURL:        apiBaseURL,
		Client:         &http.Client{Transport: tr},
		mu:             &sync.Mutex{},
	}
	var err error
	t.appsTransport, err = NewAppsTransport(t.tr, t.integrationID, privateKey)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// RoundTrip implements http.RoundTripper interface.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.Token()
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Add("Accept", acceptHeader) // We add to "Accept" header to avoid overwriting existing req headers.
	resp, err := t.tr.RoundTrip(req)
	return resp, err
}

// Token checks the active token expiration and renews if necessary. Token returns
// a valid access token. If renewal fails an error is returned.
func (t *Transport) Token() (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.token == nil || t.token.ExpiresAt.Add(-time.Minute).Before(time.Now()) {
		// Token is not set or expired/nearly expired, so refresh
		if err := t.refreshToken(); err != nil {
			return "", fmt.Errorf("could not refresh installation id %v's token: %s", t.installationID, err)
		}
	}

	return t.token.Token, nil
}

func (t *Transport) refreshToken() error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/installations/%v/access_tokens", t.BaseURL, t.installationID), nil)
	if err != nil {
		return fmt.Errorf("could not create request: %s", err)
	}

	t.appsTransport.BaseURL = t.BaseURL
	t.appsTransport.Client = t.Client
	resp, err := t.appsTransport.RoundTrip(req)
	if err != nil {
		return fmt.Errorf("could not get access_tokens from GitHub API for installation ID %v: %v", t.installationID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("received non 2xx response status %q when fetching %v", resp.Status, req.URL)
	}

	if err := json.NewDecoder(resp.Body).Decode(&t.token); err != nil {
		return err
	}

	return nil
}
