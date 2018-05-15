package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
)

type NewGitHubClientOption struct {
	// Required
	PrivateKey []byte
	// Required
	IntegrationID int

	// Either InstallationID OR (RepoOwner AND RepoName) is required.
	InstallationID int
	RepoOwner      string
	RepoName       string

	// Optional
	Client *http.Client
}

func NewGitHubClient(ctx context.Context, opt *NewGitHubClientOption) (*github.Client, error) {
	client := opt.Client
	if client == nil {
		client = http.DefaultClient
	}
	installationID, err := installationIDFromOpt(ctx, opt)
	if err != nil {
		return nil, err
	}
	itr, err := ghinstallation.New(client.Transport, opt.IntegrationID, installationID, opt.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create gh transport: %v", err)
	}
	client.Transport = itr
	return github.NewClient(client), nil
}

func installationIDFromOpt(ctx context.Context, opt *NewGitHubClientOption) (int, error) {
	if opt.InstallationID != 0 {
		return opt.InstallationID, nil
	}
	if opt.RepoOwner == "" || opt.RepoName == "" {
		return 0, errors.New("both repo owner and repo name are required")
	}
	repoFullName := fmt.Sprintf("%s/%s", opt.RepoOwner, opt.RepoName)
	ok, installation, err := getInstallation(ctx, repoFullName)
	if err != nil {
		return 0, fmt.Errorf("failed to get installation: %v", err)
	}
	if !ok {
		return 0, errors.New("installation ID not found")
	}
	return installation.InstallationID, nil
}
