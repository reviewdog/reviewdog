package server

import (
	"context"
	"io/ioutil"
	"time"

	"github.com/google/go-github/v38/github"
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
	// Retry requests because GitHub API somehow returns 401 Bad credentials from
	// time to time...
	var err error
	for i := 0; i < 5; i++ {
		checkRun, resp, err1 := c.Checks.UpdateCheckRun(ctx, owner, repo, checkID, opt)
		if err1 != nil {
			err = err1
			b, err1 := ioutil.ReadAll(resp.Body)
			if err1 != nil {
				aelog.Errorf(ctx, "failed to read error response body: %v", err1)
			}
			aelog.Errorf(ctx, "UpdateCheckRun failed: %s", string(b))
			aelog.Debugf(ctx, "Retrying UpdateCheckRun...: %d", i+1)
			time.Sleep(time.Second)
			continue
		}
		return checkRun, nil
	}

	return nil, err
}
