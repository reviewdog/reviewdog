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

// gitHubListFilesCap is GitHub's documented cap on how many files the "list
// pull request files" API will return for a single pull request, with no
// indication in the response when a PR exceeds it. See
// https://docs.github.com/en/rest/pulls/pulls?apiVersion=2022-11-28#list-pull-requests-files
const gitHubListFilesCap = 3000

// Diff returns a diff of PullRequest.
func (p *PullRequestDiffService) Diff(ctx context.Context) ([]byte, error) {
	opt := github.RawOptions{Type: github.Diff}
	d, resp, err := p.Cli.PullRequests.GetRaw(ctx, p.Owner, p.Repo, p.PR, opt)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotAcceptable {
			// GitHub refuses to generate a unified diff for some pull requests
			// (most commonly ones that changed more than 300 files, see
			// https://github.com/reviewdog/reviewdog/issues/2150) and returns
			// 406 Not Acceptable instead of a diff.
			if p.FallBackToGitCLI {
				// Prefer the local git CLI, as before the files-API fallback
				// below existed: it has no file-count limit of its own, and
				// correctly represents a pure rename/copy with no content
				// change without needing the special-casing that the
				// files-API reconstruction below requires. Only fall back to
				// the files API when there's no local repository to diff
				// against in the first place (below).
				log.Print("reviewdog: fallback to use git command")
				return p.diffUsingGitCommand(ctx)
			}
			// No local repository to diff against (e.g. the hosted doghouse
			// server, which sets FallBackToGitCLI: false): rebuild an
			// equivalent diff from the paginated "list pull request files"
			// API instead, which has no such limit of its own (though see
			// gitHubListFilesCap above).
			if fd, ferr := p.diffUsingFilesAPI(ctx); ferr == nil {
				return fd, nil
			} else {
				// Don't let this failure hide behind the original 406: on
				// doghouse in particular there's no other fallback left, so
				// whatever caused it (rate limit, a token missing a scope,
				// etc.) is otherwise invisible.
				log.Printf("reviewdog: pull request files API fallback also failed: %v", ferr)
			}
		}

		return nil, err
	}
	return []byte(d), nil
}

// diffUsingFilesAPI reconstructs a unified diff of the pull request from the
// paginated "list pull request files" API (one CommitFile per changed file,
// each carrying its own patch/hunk text). Unlike the diff API used by Diff,
// this API doesn't refuse to serve pull requests with a lot of changed files
// -- up to gitHubListFilesCap, at least; see the check below. GitHub still
// omits the per-file patch for individual files that are binary or whose own
// diff is too large; such files are silently skipped since there is no
// line-level diff to reconstruct for them anyway. A renamed or copied file
// with no content change also has no patch, but unlike a binary/oversized
// file it's still emitted: as a hunk-less entry carrying only the "diff
// --git" extended header, the same way `git diff --find-renames` represents
// a 100%-similarity rename/copy.
func (p *PullRequestDiffService) diffUsingFilesAPI(ctx context.Context) ([]byte, error) {
	var buf bytes.Buffer
	opts := &github.ListOptions{PerPage: 100}
	total := 0
	for {
		files, resp, err := p.Cli.PullRequests.ListFiles(ctx, p.Owner, p.Repo, p.PR, opts)
		if err != nil {
			return nil, err
		}
		total += len(files)
		for _, f := range files {
			status := f.GetStatus()
			patch := f.GetPatch()
			isRenameOrCopy := status == "renamed" || status == "copied"
			if patch == "" && !isRenameOrCopy {
				// Binary files (or per-file diffs too large on their own)
				// come back with no Patch at all; there's no line-level diff
				// to reconstruct for them, so skip them entirely.
				continue
			}

			name := f.GetFilename()
			oldName := name
			if isRenameOrCopy {
				if prev := f.GetPreviousFilename(); prev != "" {
					oldName = prev
				}
			}
			oldPath := "a/" + oldName
			newPath := "b/" + name
			switch status {
			case "added":
				oldPath = "/dev/null"
			case "removed":
				newPath = "/dev/null"
			}

			// Precede every entry with its own "diff --git" line. This
			// matters most for the no-patch rename/copy case just below:
			// diff/parse.go only accepts a file diff with an empty hunk list
			// when whatever follows starts with "diff" (that's the existing
			// "file diff with empty hunks, e.g. deleting an empty file"
			// case) -- and it treats running out of input the same way, so
			// this also covers such an entry being the very last one in the
			// whole reconstructed diff. Emitting this line unconditionally
			// keeps every entry self-delimiting regardless of what precedes
			// or follows it.
			fmt.Fprintf(&buf, "diff --git %s %s\n", oldPath, newPath)

			if patch == "" {
				// Pure rename/copy, no content change: no ---/+++ or hunks,
				// same as real `git diff` output for a 100%-similarity
				// rename/copy. Filenames containing a literal tab character
				// would still confuse the "---"/"+++" header parser (it
				// splits on the last tab as a timestamp separator) if they
				// went through that path, but skipping ---/+++ here sides
				// steps that for this case; it remains a known, unfixed
				// limitation for the patched case just below.
				continue
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
	if total >= gitHubListFilesCap {
		log.Printf("reviewdog: pull request #%d returned %d changed files from the files API, at or beyond GitHub's documented %d-file cap for this endpoint with no indication whether more exist; the reconstructed diff may be missing files", p.PR, total, gitHubListFilesCap)
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
