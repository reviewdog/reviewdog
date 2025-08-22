package gitea

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"code.gitea.io/sdk/gitea"
	"github.com/reviewdog/reviewdog"
)

var _ reviewdog.DiffService = (*PullRequestDiffService)(nil)

// PullRequestDiffService is a DiffService which uses Gitea Diff API.
type PullRequestDiffService struct {
	Cli              *gitea.Client
	Owner            string
	Repo             string
	PR               int64
	SHA              string
	FallBackToGitCLI bool
}

// Strip returns 1 as a strip of git diff.
func (p *PullRequestDiffService) Strip() int {
	return 1
}

// Diff returns a diff of PullRequest.
func (p *PullRequestDiffService) Diff(ctx context.Context) ([]byte, error) {
	d, resp, err := p.Cli.GetPullRequestDiff(p.Owner, p.Repo, p.PR, gitea.PullRequestDiffOptions{
		Binary: false,
	})
	if err != nil {
		if resp != nil && p.FallBackToGitCLI && resp.StatusCode == http.StatusNotFound {
			log.Print("reviewdog: fallback to use git command")
			return p.diffUsingGitCommand(ctx)
		}

		return nil, err
	}
	return []byte(d), nil
}

// diffUsingGitCommand returns a diff of PullRequest using git command.
func (p *PullRequestDiffService) diffUsingGitCommand(ctx context.Context) ([]byte, error) {
	pr, _, err := p.Cli.GetPullRequest(p.Owner, p.Repo, p.PR)
	if err != nil {
		return nil, err
	}

	head := pr.Head
	headSha := head.Sha

	if os.Getenv("REVIEWDOG_SKIP_GIT_FETCH") != "true" {
		for _, sha := range []string{pr.Base.Sha, headSha} {
			bytes, err := exec.Command("git", "fetch", "--depth=1", head.Repository.HTMLURL, sha).CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("failed to run git fetch: %s\n%w", bytes, err)
			}
		}
	}

	bytes, err := exec.Command("git", "diff", "--find-renames", pr.Base.Sha, headSha).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run git diff: %s\n%w", bytes, err)
	}

	return bytes, nil
}
