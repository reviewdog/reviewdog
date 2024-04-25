package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/reviewdog/reviewdog/doghouse"
	"golang.org/x/oauth2"
)

const notokenSkipTestMes = "skipping test (requires actual Personal access tokens. export REVIEWDOG_TEST_GITHUB_API_TOKEN=<GitHub Personal Access Token>)"

func setupGitHubClient() *github.Client {
	token := os.Getenv("REVIEWDOG_TEST_GITHUB_API_TOKEN")
	if token == "" {
		return nil
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.TODO(), ts)
	return github.NewClient(tc)
}

func TestChecker_GetPullRequestDiff(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test which contains actual API requests in short mode")
	}
	client := setupGitHubClient()
	if client == nil {
		t.Skip(notokenSkipTestMes)
	}

	want := `diff --git a/.codecov.yml b/.codecov.yml
index aa49124774..781ee2492f 100644
--- a/.codecov.yml
+++ b/.codecov.yml
@@ -7,5 +7,4 @@ coverage:
       default:
         target: 0%
 
-comment:
-  layout: "header"
+comment: false
`

	// https://github.com/reviewdog/reviewdog/pull/73
	owner := "haya14busa"
	repo := "reviewdog"
	pr := 73
	b, err := NewChecker(&doghouse.CheckRequest{}, client).gh.GetPullRequestDiff(context.Background(), owner, repo, pr)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != want {
		t.Errorf("got:\n%v\nwant:\n%v", got, want)
	}
}

func TestChecker_GetPullRequestDiff_fake(t *testing.T) {
	apiCalled := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/pulls/14", func(w http.ResponseWriter, r *http.Request) {
		apiCalled++
		if r.Method != http.MethodGet {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		if accept := r.Header.Get("Accept"); !strings.Contains(accept, "diff") {
			t.Errorf("Accept header doesn't contain 'diff': %v", accept)
		}
		w.Write([]byte("Pull Request diff"))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")

	gh := NewChecker(&doghouse.CheckRequest{}, cli).gh

	if _, err := gh.GetPullRequestDiff(context.Background(), "o", "r", 14); err != nil {
		t.Fatal(err)
	}
	if apiCalled != 1 {
		t.Errorf("GitHub API should be called once; called %v times", apiCalled)
	}
}

func TestChecker_GetPullRequestDiff_fake_fallback(t *testing.T) {
	apiCalled := 0
	mux := http.NewServeMux()
	headSHA := "HEAD^"
	baseSHA := "HEAD"
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

	t.Setenv("REVIEWDOG_SKIP_GIT_FETCH", "true")

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")

	gh := NewChecker(&doghouse.CheckRequest{}, cli).gh

	if _, err := gh.GetPullRequestDiff(context.Background(), "o", "r", 14); err != nil {
		t.Fatal(err)
	}
	if apiCalled != 2 {
		t.Errorf("GitHub API should be called twice; called %v times", apiCalled)
	}
}
