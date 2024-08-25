package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-github/v64/github"
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

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")

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

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")

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
