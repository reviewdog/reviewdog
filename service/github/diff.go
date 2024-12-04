package github

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/google/go-github/v64/github"
	"github.com/reviewdog/reviewdog"
)

var _ reviewdog.DiffService = (*PullRequestDiffService)(nil)

// PullRequestDiffService is a DiffService which uses GitHub Diff API.
type PullRequestDiffService struct {
	Cli              *github.Client
	Owner            string
	Repo             string
	PR               int
	SHA              string
	FallBackToGitCLI bool
}

// Strip returns 1 as a strip of git diff.
func (p *PullRequestDiffService) Strip() int {
	return 1
}

// Diff returns a diff of PullRequest.
func (p *PullRequestDiffService) Diff(ctx context.Context) ([]byte, error) {
	opt := github.RawOptions{Type: github.Diff}
	d, resp, err := p.Cli.PullRequests.GetRaw(ctx, p.Owner, p.Repo, p.PR, opt)
	if err != nil {
		if resp != nil && p.FallBackToGitCLI && resp.StatusCode == http.StatusNotAcceptable {
			log.Print("reviewdog: fallback to use git command")
			return p.diffUsingGitCommand(ctx)
		}

		return nil, err
	}
	return []byte(d), nil
}

// diffUsingGitCommand returns a diff of PullRequest using git command.
func (p *PullRequestDiffService) diffUsingGitCommand(ctx context.Context) ([]byte, error) {
	pr, _, err := p.Cli.PullRequests.Get(ctx, p.Owner, p.Repo, p.PR)
	if err != nil {
		return nil, err
	}

	head := pr.GetHead()
	headSha := head.GetSHA()

	commitsComparison, _, err := p.Cli.Repositories.CompareCommits(ctx, p.Owner, p.Repo, headSha, pr.GetBase().GetSHA(), nil)
	if err != nil {
		return nil, err
	}

	mergeBaseSha := commitsComparison.GetMergeBaseCommit().GetSHA()

	if os.Getenv("REVIEWDOG_SKIP_GIT_FETCH") != "true" {
		for _, sha := range []string{mergeBaseSha, headSha} {
			bytes, err := exec.Command("git", "fetch", "--depth=1", head.GetRepo().GetHTMLURL(), sha).CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("failed to run git fetch: %s\n%w", bytes, err)
			}
		}
	}

	bytes, err := exec.Command("git", "diff", "--find-renames", mergeBaseSha, headSha).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run git diff: %s\n%w", bytes, err)
	}

	return bytes, nil
}
