package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-github/v92/github"
	"github.com/reviewdog/reviewdog/diff"
)

const sampleDiff = `--- a/sample.old.txt	2016-10-13 05:09:35.820791185 +0900
+++ b/sample.new.txt	2016-10-13 05:15:26.839245048 +0900
@@ -1,3 +1,4 @@
 unchanged, contextual line
-deleted line
+added line
+added line
 unchanged, contextual line
--- a/nonewline.old.txt	2016-10-13 15:34:14.931778318 +0900
+++ b/nonewline.new.txt	2016-10-13 15:34:14.868444672 +0900
@@ -1,4 +1,4 @@
 " vim: nofixeol noendofline
 No newline at end of both the old and new file
-a
-a
\ No newline at end of file
+b
+b
\ No newline at end of file
`

func TestDiff_success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/pulls/14", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleDiff))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := newGitHubClient(t, ts.URL)

	diffService := &PullRequestDiffService{
		Cli:              cli,
		Owner:            "o",
		Repo:             "r",
		PR:               14,
		SHA:              "sha",
		FallBackToGitCLI: false,
	}

	_, err := diffService.Diff(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestDiff_fallbackToGitCli(t *testing.T) {
	t.Setenv("REVIEWDOG_SKIP_GIT_FETCH", "true")
	apiCalled := 0
	headSHA := "HEAD^"
	baseSHA := "HEAD"
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/pulls/14", func(w http.ResponseWriter, r *http.Request) {
		apiCalled++
		if r.Method != http.MethodGet {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		if accept := r.Header.Get("Accept"); strings.Contains(accept, "diff") {
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}
		if accept := r.Header.Get("Accept"); accept != "application/vnd.github.v3+json" {
			t.Errorf("Accept header doesn't contain 'diff': %v", accept)
		}

		pullRequestJSON, err := json.Marshal(github.PullRequest{
			Head: &github.PullRequestBranch{
				SHA: &headSHA,
			},
			Base: &github.PullRequestBranch{
				SHA: &baseSHA,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := w.Write(pullRequestJSON); err != nil {
			t.Fatal(err)
		}
	})
	mux.HandleFunc("/repos/o/r/compare/"+headSHA+"..."+baseSHA, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		if accept := r.Header.Get("Accept"); accept != "application/vnd.github.v3+json" {
			t.Errorf("Accept header doesn't contain 'diff': %v", accept)
		}

		mergeBaseSha := "HEAD^"

		commitsComparisonJSON, err := json.Marshal(github.CommitsComparison{
			MergeBaseCommit: &github.RepositoryCommit{
				SHA: &mergeBaseSha,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := w.Write(commitsComparisonJSON); err != nil {
			t.Fatal(err)
		}
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := newGitHubClient(t, ts.URL)

	diffService := &PullRequestDiffService{
		Cli:              cli,
		Owner:            "o",
		Repo:             "r",
		PR:               14,
		SHA:              "sha",
		FallBackToGitCLI: true,
	}

	_, err := diffService.Diff(context.Background())
	if err != nil {
		t.Error(err)
	}
	if apiCalled != 2 {
		t.Errorf("GitHub API should be called twice; called %v times", apiCalled)
	}
}

// TestDiff_fallbackToFilesAPI reproduces the scenario reported in
// https://github.com/reviewdog/reviewdog/issues/2150: GitHub's diff API
// returns 406 Not Acceptable for pull requests with a lot of changed files,
// and FallBackToGitCLI is false (as it is on the hosted doghouse server,
// which has no local repository to run `git diff` against). Without a
// files-API-based fallback, Diff would just return the raw 406 error.
func TestDiff_fallbackToFilesAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/pulls/14", func(w http.ResponseWriter, r *http.Request) {
		if accept := r.Header.Get("Accept"); strings.Contains(accept, "diff") {
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}
		t.Errorf("unexpected non-diff request to /pulls/14: %v", r.URL)
	})
	mux.HandleFunc("/repos/o/r/pulls/14/files", func(w http.ResponseWriter, r *http.Request) {
		files := []*github.CommitFile{
			{
				Filename: new("added.txt"),
				Status:   new("added"),
				Patch:    new("@@ -0,0 +1,2 @@\n+line one\n+line two"),
			},
			{
				Filename: new("removed.txt"),
				Status:   new("removed"),
				Patch:    new("@@ -1,2 +0,0 @@\n-old one\n-old two"),
			},
			{
				Filename:         new("renamed_new.txt"),
				PreviousFilename: new("renamed_old.txt"),
				Status:           new("renamed"),
				Patch:            new("@@ -1 +1 @@\n-old content\n+new content"),
			},
			{
				// Binary files (or per-file diffs too large on their own)
				// come back with no Patch at all; they must be skipped
				// rather than emitted as a bogus empty hunk.
				Filename: new("image.png"),
				Status:   new("modified"),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(files); err != nil {
			t.Fatal(err)
		}
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := newGitHubClient(t, ts.URL)

	diffService := &PullRequestDiffService{
		Cli:              cli,
		Owner:            "o",
		Repo:             "r",
		PR:               14,
		SHA:              "sha",
		FallBackToGitCLI: false,
	}

	got, err := diffService.Diff(context.Background())
	if err != nil {
		t.Fatalf("Diff() returned an error, want a diff reconstructed from the files API: %v", err)
	}

	fds, err := diff.ParseMultiFile(bytes.NewReader(got))
	if err != nil {
		t.Fatalf("failed to parse reconstructed diff: %v\n---\n%s\n---", err, got)
	}
	if len(fds) != 3 {
		t.Fatalf("got %d file diffs, want 3 (image.png must be skipped, not counted):\n%s", len(fds), got)
	}

	if got, want := fds[0].PathOld, "/dev/null"; got != want {
		t.Errorf("added.txt: PathOld = %q, want %q", got, want)
	}
	if got, want := fds[0].PathNew, "b/added.txt"; got != want {
		t.Errorf("added.txt: PathNew = %q, want %q", got, want)
	}

	if got, want := fds[1].PathOld, "a/removed.txt"; got != want {
		t.Errorf("removed.txt: PathOld = %q, want %q", got, want)
	}
	if got, want := fds[1].PathNew, "/dev/null"; got != want {
		t.Errorf("removed.txt: PathNew = %q, want %q", got, want)
	}

	if got, want := fds[2].PathOld, "a/renamed_old.txt"; got != want {
		t.Errorf("renamed: PathOld = %q, want %q", got, want)
	}
	if got, want := fds[2].PathNew, "b/renamed_new.txt"; got != want {
		t.Errorf("renamed: PathNew = %q, want %q", got, want)
	}
}

// TestDiff_fallbackToFilesAPI_pagination exercises the pagination loop in
// diffUsingFilesAPI (opts.Page = resp.NextPage): the scenario this PR
// targets, a pull request with enough changed files to need more than one
// page from the "list pull request files" API, was not covered by
// TestDiff_fallbackToFilesAPI, whose mock always served every file on a
// single page regardless of the requested page number.
func TestDiff_fallbackToFilesAPI_pagination(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/pulls/20", func(w http.ResponseWriter, r *http.Request) {
		if accept := r.Header.Get("Accept"); strings.Contains(accept, "diff") {
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}
		t.Errorf("unexpected non-diff request to /pulls/20: %v", r.URL)
	})

	var pagesRequested []string
	mux.HandleFunc("/repos/o/r/pulls/20/files", func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		pagesRequested = append(pagesRequested, page)
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/o/r/pulls/20/files?page=2>; rel="next"`, r.Host))
			if err := json.NewEncoder(w).Encode([]*github.CommitFile{
				{Filename: new("a.txt"), Status: new("added"), Patch: new("@@ -0,0 +1,1 @@\n+line a")},
			}); err != nil {
				t.Fatal(err)
			}
		case "2":
			if err := json.NewEncoder(w).Encode([]*github.CommitFile{
				{Filename: new("b.txt"), Status: new("added"), Patch: new("@@ -0,0 +1,1 @@\n+line b")},
			}); err != nil {
				t.Fatal(err)
			}
		default:
			t.Errorf("unexpected page request: %v", page)
		}
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := newGitHubClient(t, ts.URL)

	diffService := &PullRequestDiffService{
		Cli:              cli,
		Owner:            "o",
		Repo:             "r",
		PR:               20,
		SHA:              "sha",
		FallBackToGitCLI: false,
	}

	got, err := diffService.Diff(context.Background())
	if err != nil {
		t.Fatalf("Diff() returned an error: %v", err)
	}
	if len(pagesRequested) != 2 {
		t.Fatalf("expected 2 page requests to the files API, got %d: %v", len(pagesRequested), pagesRequested)
	}

	fds, err := diff.ParseMultiFile(bytes.NewReader(got))
	if err != nil {
		t.Fatalf("failed to parse reconstructed diff: %v\n---\n%s\n---", err, got)
	}
	if len(fds) != 2 {
		t.Fatalf("got %d file diffs, want 2 (one per page):\n%s", len(fds), got)
	}
	if got, want := fds[0].PathNew, "b/a.txt"; got != want {
		t.Errorf("page 1 file: PathNew = %q, want %q", got, want)
	}
	if got, want := fds[1].PathNew, "b/b.txt"; got != want {
		t.Errorf("page 2 file: PathNew = %q, want %q", got, want)
	}
}

// TestDiff_fallbackToFilesAPI_renameAndCopyWithoutContentChange covers three
// gaps flagged in review on this PR: a pure rename with no content change
// was silently dropped from the reconstructed diff instead of appearing as a
// hunk-less entry (as `git diff --find-renames` would show it); GitHub's
// "copied" status wasn't handled at all, falling into the same bucket as a
// plain modification and using the file's own name instead of
// PreviousFilename for the old path; and the no-content-change case was only
// tried mid-stream, not as the very last file in the reconstructed diff
// (which exercises a different, EOF-based code path in diff/parse.go).
func TestDiff_fallbackToFilesAPI_renameAndCopyWithoutContentChange(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/pulls/21", func(w http.ResponseWriter, r *http.Request) {
		if accept := r.Header.Get("Accept"); strings.Contains(accept, "diff") {
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}
		t.Errorf("unexpected non-diff request to /pulls/21: %v", r.URL)
	})
	mux.HandleFunc("/repos/o/r/pulls/21/files", func(w http.ResponseWriter, r *http.Request) {
		files := []*github.CommitFile{
			{
				// Pure rename, no content change: mid-stream (followed by
				// more files below).
				Filename:         new("renamed_new.txt"),
				PreviousFilename: new("renamed_old.txt"),
				Status:           new("renamed"),
			},
			{
				Filename:         new("copied_new.txt"),
				PreviousFilename: new("copied_old.txt"),
				Status:           new("copied"),
				Patch:            new("@@ -0,0 +1,1 @@\n+line one"),
			},
			{
				// Pure copy, no content change: the very last file, which
				// exercises the EOF path rather than the mid-stream one
				// above.
				Filename:         new("copied_new2.txt"),
				PreviousFilename: new("copied_old2.txt"),
				Status:           new("copied"),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(files); err != nil {
			t.Fatal(err)
		}
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := newGitHubClient(t, ts.URL)

	diffService := &PullRequestDiffService{
		Cli:              cli,
		Owner:            "o",
		Repo:             "r",
		PR:               21,
		SHA:              "sha",
		FallBackToGitCLI: false,
	}

	got, err := diffService.Diff(context.Background())
	if err != nil {
		t.Fatalf("Diff() returned an error: %v", err)
	}

	fds, err := diff.ParseMultiFile(bytes.NewReader(got))
	if err != nil {
		t.Fatalf("failed to parse reconstructed diff: %v\n---\n%s\n---", err, got)
	}
	if len(fds) != 3 {
		t.Fatalf("got %d file diffs, want 3:\n%s", len(fds), got)
	}

	if got, want := len(fds[0].Hunks), 0; got != want {
		t.Errorf("renamed_new.txt (no content change): got %d hunks, want %d", got, want)
	}

	if got, want := fds[1].PathOld, "a/copied_old.txt"; got != want {
		t.Errorf("copied_new.txt: PathOld = %q, want %q (not the file's own name)", got, want)
	}
	if got, want := fds[1].PathNew, "b/copied_new.txt"; got != want {
		t.Errorf("copied_new.txt: PathNew = %q, want %q", got, want)
	}
	if got, want := len(fds[1].Hunks), 1; got != want {
		t.Errorf("copied_new.txt: got %d hunks, want %d", got, want)
	}

	if got, want := len(fds[2].Hunks), 0; got != want {
		t.Errorf("copied_new2.txt (no content change, last file): got %d hunks, want %d", got, want)
	}
}
