package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/google/go-github/v60/github"
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
	d, resp, err := c.PullRequests.GetRaw(ctx, owner, repo, number, opt)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotAcceptable && c.checkInstallGitCommand() {
			log.Print("fallback to use git command")
			return c.getPullRequestDiffUsingGitCommand(ctx, owner, repo, number)
		}

		return nil, err
	}
	return []byte(d), err
}

// checkInstallGitCommand checks if git command is installed.
func (c *checkerGitHubClient) checkInstallGitCommand() bool {
	_, err := exec.Command("git", "-v").CombinedOutput()
	return err == nil
}

// getPullRequestDiffUsingGitCommand returns a diff of PullRequest using git command.
func (c *checkerGitHubClient) getPullRequestDiffUsingGitCommand(ctx context.Context, owner, repo string, number int) ([]byte, error) {
	pr, _, err := c.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, err
	}

	head := pr.GetHead()
	headSha := head.GetSHA()

	commitsComparison, _, err := c.Repositories.CompareCommits(ctx, owner, repo, headSha, pr.GetBase().GetSHA(), nil)
	if err != nil {
		return nil, err
	}

	mergeBaseSha := commitsComparison.GetMergeBaseCommit().GetSHA()

	if os.Getenv("REVIEWDOG_SKIP_GIT_FETCH") != "true" {
		for _, sha := range []string{mergeBaseSha, headSha} {
			_, err := exec.Command("git", "fetch", "--depth=1", head.GetRepo().GetHTMLURL(), sha).CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("failed to run git fetch: %w", err)
			}
		}
	}

	bytes, err := exec.Command("git", "diff", "--find-renames", mergeBaseSha, headSha).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run git diff: %w", err)
	}

	return bytes, nil
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
			b, err1 := io.ReadAll(resp.Body)
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
