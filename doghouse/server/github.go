package server

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v57/github"
)

type NewGitHubClientOption struct {
	// Required
	PrivateKey []byte
	// Required
	IntegrationID int

	// RepoOwner is required for installation API.
	RepoOwner string

	// Optional
	Client *http.Client
}

func NewGitHubClient(ctx context.Context, opt *NewGitHubClientOption) (*github.Client, error) {
	client := opt.Client
	if client == nil {
		client = http.DefaultClient
	}

	itr, err := githubAppTransport(ctx, client, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create gh transport: %w", err)
	}

	client.Transport = itr

	ghcli := github.NewClient(client)
	if IsGithubEnterpriseApi() {
		url := GetGithubHostUrl()
		ghcli, err = ghcli.WithEnterpriseURLs(url, url)
	}
	return ghcli, nil
}

func githubAppTransport(ctx context.Context, client *http.Client, opt *NewGitHubClientOption) (http.RoundTripper, error) {
	if opt.RepoOwner == "" {
		transport, err := ghinstallation.NewAppsTransport(getTransport(client), int64(opt.IntegrationID), opt.PrivateKey)
		transport.BaseURL = GetGithubApiUrl()
		return transport, err
	}
	installationID, err := findInstallationID(ctx, opt)
	if err != nil {
		return nil, err
	}
	transport, err := ghinstallation.New(getTransport(client), int64(opt.IntegrationID), installationID, opt.PrivateKey)
	transport.BaseURL = GetGithubApiUrl()
	return transport, err
}

func getTransport(client *http.Client) http.RoundTripper {
	if client.Transport != nil {
		return client.Transport
	}
	return http.DefaultTransport
}

func findInstallationID(ctx context.Context, opt *NewGitHubClientOption) (int64, error) {
	appCli, err := NewGitHubClient(ctx, &NewGitHubClientOption{
		PrivateKey:    opt.PrivateKey,
		IntegrationID: opt.IntegrationID,
		Client:        &http.Client{}, // Use different client to get installation.
		// Do no set RepoOwner.
	})
	if err != nil {
		return 0, err
	}
	inst, _, err := appCli.Apps.FindUserInstallation(ctx, opt.RepoOwner)
	if err != nil {
		return 0, err
	}
	return inst.GetID(), nil
}

func getBaseEnterpriseUrl() string {
	return os.Getenv("GITHUB_ENTERPRISE_BASE_URL")
}

func IsGithubEnterpriseApi() bool {
	return getBaseEnterpriseUrl() != ""
}

// GetGithubHostUrl Used for login methods, that not directly related to GitHub API.
func GetGithubHostUrl() string {
	enterpriseUrl := getBaseEnterpriseUrl()

	if enterpriseUrl != "" {
		return enterpriseUrl
	} else {
		return "https://github.com"
	}
}

func GetGithubApiUrl() string {
	enterpriseUrl := getBaseEnterpriseUrl()

	if enterpriseUrl != "" {
		return enterpriseUrl + "/api/v3"
	} else {
		return "https://api.github.com"
	}
}
