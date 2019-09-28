package ghinstallation

import (
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// AppsTransport provides a http.RoundTripper by wrapping an existing
// http.RoundTripper and provides GitHub Apps authentication as a
// GitHub App.
//
// Client can also be overwritten, and is useful to change to one which
// provides retry logic if you do experience retryable errors.
//
// See https://developer.github.com/apps/building-integrations/setting-up-and-registering-github-apps/about-authentication-options-for-github-apps/
type AppsTransport struct {
	BaseURL       string            // BaseURL is the scheme and host for GitHub API, defaults to https://api.github.com
	Client        Client            // Client to use to refresh tokens, defaults to http.Client with provided transport
	tr            http.RoundTripper // tr is the underlying roundtripper being wrapped
	key           *rsa.PrivateKey   // key is the GitHub Integration's private key
	integrationID int               // integrationID is the GitHub Integration's Installation ID
}

// NewAppsTransportKeyFromFile returns a AppsTransport using a private key from file.
func NewAppsTransportKeyFromFile(tr http.RoundTripper, integrationID int, privateKeyFile string) (*AppsTransport, error) {
	privateKey, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("could not read private key: %s", err)
	}
	return NewAppsTransport(tr, integrationID, privateKey)
}

// NewAppsTransport returns a AppsTransport using private key. The key is parsed
// and if any errors occur the error is non-nil.
//
// The provided tr http.RoundTripper should be shared between multiple
// installations to ensure reuse of underlying TCP connections.
//
// The returned Transport's RoundTrip method is safe to be used concurrently.
func NewAppsTransport(tr http.RoundTripper, integrationID int, privateKey []byte) (*AppsTransport, error) {
	t := &AppsTransport{
		tr:            tr,
		integrationID: integrationID,
		BaseURL:       apiBaseURL,
		Client:        &http.Client{Transport: tr},
	}
	var err error
	t.key, err = jwt.ParseRSAPrivateKeyFromPEM(privateKey)
	if err != nil {
		return nil, fmt.Errorf("could not parse private key: %s", err)
	}
	return t, nil
}

// RoundTrip implements http.RoundTripper interface.
func (t *AppsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	claims := &jwt.StandardClaims{
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
		Issuer:    strconv.Itoa(t.integrationID),
	}
	bearer := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	ss, err := bearer.SignedString(t.key)
	if err != nil {
		return nil, fmt.Errorf("could not sign jwt: %s", err)
	}

	req.Header.Set("Authorization", "Bearer "+ss)
	req.Header.Set("Accept", acceptHeader)

	resp, err := t.tr.RoundTrip(req)
	return resp, err
}
