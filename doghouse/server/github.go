package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v26/github"
	"github.com/reviewdog/reviewdog/doghouse/server/storage"
)

type NewGitHubClientOption struct {
	// Required
	PrivateKey []byte
	// Required
	IntegrationID int

	// RepoOwner AND InstallationStore is required for installation API.
	RepoOwner         string
	InstallationStore storage.GitHubInstallationStore

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
		return nil, fmt.Errorf("failed to create gh transport: %v", err)
	}

	client.Transport = itr
	return github.NewClient(client), nil
}

func githubAppTransport(ctx context.Context, client *http.Client, opt *NewGitHubClientOption) (http.RoundTripper, error) {
	if opt.RepoOwner == "" {
		return ghinstallation.NewAppsTransport(client.Transport, opt.IntegrationID, opt.PrivateKey)
	}

	installationID, err := installationIDFromOpt(ctx, opt)
	if err != nil {
		return nil, err
	}
	return ghinstallation.New(client.Transport, opt.IntegrationID, int(installationID), opt.PrivateKey)
}

func installationIDFromOpt(ctx context.Context, opt *NewGitHubClientOption) (int64, error) {
	if opt.RepoOwner == "" {
		return 0, errors.New("repo owner is required")
	}
	if opt.InstallationStore == nil {
		return 0, errors.New("instllation store is not provided")
	}
	ok, inst, err := opt.InstallationStore.Get(ctx, opt.RepoOwner)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve installation ID: %v", err)
	}
	if !ok {
		return 0, fmt.Errorf("installation ID not found for %s", opt.RepoOwner)
	}
	return inst.InstallationID, nil
}
