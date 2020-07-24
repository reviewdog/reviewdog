package server

import (
	"context"
	"io/ioutil"

	"github.com/google/go-github/v32/github"
	"github.com/vvakame/sdlog/aelog"
)

type checkerGitHubClientInterface interface {
	GetPullRequestDiff(ctx context.Context, owner, repo string, number int) ([]byte, error)
	CreateCheckRun(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error)
	UpdateCheckRun(ctx context.Context, owner, repo string, checkID int64, opt github.UpdateCheckRunOptions) (*github.CheckRun, error)
}

type checkerGitHubClient struct {
	*github.Client
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

func (c *checkerGitHubClient) UpdateCheckRun(ctx context.Context, owner, repo string, checkID int64, opt github.UpdateCheckRunOptions) (*github.CheckRun, error) {
	checkRun, resp, err := c.Checks.UpdateCheckRun(ctx, owner, repo, checkID, opt)
	if err != nil {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			aelog.Errorf(ctx, "failed to read error response body: %v", err)
		}
		aelog.Errorf(ctx, "UpdateCheckRun failed: %s", string(b))
	}
	return checkRun, err
}
