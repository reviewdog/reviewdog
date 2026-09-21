package github

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/v92/github"
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
		if resp != nil && resp.StatusCode == http.StatusNotAcceptable {
			// GitHub refuses to generate a unified diff for some pull requests
			// (most commonly ones that changed more than 300 files, see
			// https://github.com/reviewdog/reviewdog/issues/2150) and returns
			// 406 Not Acceptable instead of a diff. Rebuild an equivalent diff
			// from the paginated "list pull request files" API, which has no
			// such limit, before falling back to a local git diff (which
			// needs repository access we might not have, e.g. when running
			// as the hosted doghouse server).
			if fd, ferr := p.diffUsingFilesAPI(ctx); ferr == nil {
				return fd, nil
			}
			if p.FallBackToGitCLI {
				log.Print("reviewdog: fallback to use git command")
				return p.diffUsingGitCommand(ctx)
			}
		}

		return nil, err
	}
	return []byte(d), nil
}

// diffUsingFilesAPI reconstructs a unified diff of the pull request from the
// paginated "list pull request files" API (one CommitFile per changed file,
// each carrying its own patch/hunk text). Unlike the diff API used by Diff,
// this API doesn't refuse to serve pull requests with a lot of changed files.
// GitHub still omits the per-file patch for individual files that are binary
// or whose own diff is too large; such files are silently skipped since there
// is no line-level diff to reconstruct for them anyway.
func (p *PullRequestDiffService) diffUsingFilesAPI(ctx context.Context) ([]byte, error) {
	var buf bytes.Buffer
	opts := &github.ListOptions{PerPage: 100}
	for {
		files, resp, err := p.Cli.PullRequests.ListFiles(ctx, p.Owner, p.Repo, p.PR, opts)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			patch := f.GetPatch()
			if patch == "" {
				continue
			}
			oldPath := "a/" + f.GetFilename()
			newPath := "b/" + f.GetFilename()
			switch f.GetStatus() {
			case "added":
				oldPath = "/dev/null"
			case "removed":
				newPath = "/dev/null"
			case "renamed":
				if prev := f.GetPreviousFilename(); prev != "" {
					oldPath = "a/" + prev
				}
			}
			fmt.Fprintf(&buf, "--- %s\n+++ %s\n%s", oldPath, newPath, patch)
			if !strings.HasSuffix(patch, "\n") {
				buf.WriteByte('\n')
			}
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	if buf.Len() == 0 {
		return nil, errors.New("reviewdog: no file with a diffable patch found via pull request files API")
	}
	return buf.Bytes(), nil
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
			// GitHub stores data in the same fork network together - i.e. you can reference
			// commits from forks using the base repo (this is how refs/pull/* works).
			// By fetching using the base repo, we can use GitHub Actions ${GITHUB_TOKEN} to authenticate
			// and fetch PR content from private forks even though the token is only scoped to the base repo.
			bytes, err := exec.Command("git", "fetch", "--depth=1", pr.GetBase().GetRepo().GetHTMLURL(), sha).CombinedOutput()
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
