package server

import (
	"context"

	"github.com/google/go-github/v26/github"
)

type checkerGitHubClientInterface interface {
	GetPullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error)
	GetPullRequestDiff(ctx context.Context, owner, repo string, number int) ([]byte, error)
	CreateCheckRun(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error)
}

type checkerGitHubClient struct {
	*github.Client
}

func (c *checkerGitHubClient) GetPullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error) {
	pr, _, err := c.PullRequests.Get(ctx, owner, repo, number)
	return pr, err
}

func (c *checkerGitHubClient) GetPullRequestDiff(ctx context.Context, owner, repo string, number int) ([]byte, error) {
	opt := github.RawOptions{Type: github.Diff}
	d, _, err := c.PullRequests.GetRaw(ctx, owner, repo, number, opt)
	return []byte(d), err
}

func (c *checkerGitHubClient) CreateCheckRun(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
	checkRun, _, err := c.Checks.CreateCheckRun(ctx, owner, repo, opt)
	return checkRun, err
}
