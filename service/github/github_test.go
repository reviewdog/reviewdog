package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/go-github/v26/github"
	"github.com/kylelemons/godebug/pretty"
	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/service/serviceutil"
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
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	return github.NewClient(tc)
}

func moveToRootDir() {
	os.Chdir("../..")
}

func TestGitHubPullRequest_Post(t *testing.T) {
	t.Skip("skipping test which post comments actually")
	client := setupGitHubClient()
	if client == nil {
		t.Skip(notokenSkipTestMes)
	}

	// https://github.com/reviewdog/reviewdog/pull/2
	owner := "haya14busa"
	repo := "reviewdog"
	pr := 2
	sha := "cce89afa9ac5519a7f5b1734db2e3aa776b138a7"

	g, err := NewGitHubPullRequest(client, owner, repo, pr, sha)
	if err != nil {
		t.Fatal(err)
	}
	comment := &reviewdog.Comment{
		CheckResult: &reviewdog.CheckResult{
			Path: "watchdogs.go",
		},
		LnumDiff: 17,
		Body:     "[reviewdog] test",
	}
	// https://github.com/reviewdog/reviewdog/pull/2/files#diff-ed1d019a10f54464cfaeaf6a736b7d27L20
	if err := g.Post(context.Background(), comment); err != nil {
		t.Error(err)
	}
	if err := g.Flush(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestGitHubPullRequest_Diff(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test which contains actual API requests in short mode")
	}
	client := setupGitHubClient()
	if client == nil {
		t.Skip(notokenSkipTestMes)
	}

	want := `diff --git a/diff.go b/diff.go
index b380b67..6abc0f1 100644
--- a/diff.go
+++ b/diff.go
@@ -4,6 +4,9 @@ import (
 	"os/exec"
 )
 
+func TestNewExportedFunc() {
+}
+
 var _ DiffService = &DiffString{}
 
 type DiffString struct {
diff --git a/reviewdog.go b/reviewdog.go
index 61450f3..f63f149 100644
--- a/reviewdog.go
+++ b/reviewdog.go
@@ -10,18 +10,18 @@ import (
 	"github.com/reviewdog/reviewdog/diff"
 )
 
+var TestExportedVarWithoutComment = 1
+
+func NewReviewdog(p Parser, c CommentService, d DiffService) *Reviewdog {
+	return &Reviewdog{p: p, c: c, d: d}
+}
+
 type Reviewdog struct {
 	p Parser
 	c CommentService
 	d DiffService
 }
 
-func NewReviewdog(p Parser, c CommentService, d DiffService) *Reviewdog {
-	return &Reviewdog{p: p, c: c, d: d}
-}
-
-// CheckResult represents a checked result of static analysis tools.
-// :h error-file-format
 type CheckResult struct {
 	Path    string   // file path
 	Lnum    int      // line number
`

	// https://github.com/reviewdog/reviewdog/pull/2
	owner := "haya14busa"
	repo := "reviewdog"
	pr := 2
	g, err := NewGitHubPullRequest(client, owner, repo, pr, "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := g.Diff(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != want {
		t.Errorf("got:\n%v\nwant:\n%v", got, want)
	}
}

func TestGitHubPullRequest_comment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test which contains actual API requests in short mode")
	}
	client := setupGitHubClient()
	if client == nil {
		t.Skip(notokenSkipTestMes)
	}
	// https://github.com/reviewdog/reviewdog/pull/2
	owner := "haya14busa"
	repo := "reviewdog"
	pr := 2
	g, err := NewGitHubPullRequest(client, owner, repo, pr, "")
	if err != nil {
		t.Fatal(err)
	}
	comments, err := g.comment(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range comments {
		t.Log("---")
		t.Log(*c.Body)
		t.Log(*c.Path)
		if c.Position != nil {
			t.Log(*c.Position)
		}
		t.Log(*c.CommitID)
	}
}

func TestGitHubPullRequest_Post_Flush_review_api(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	moveToRootDir()

	listCommentsAPICalled := 0
	postCommentsAPICalled := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/pulls/14/comments", func(w http.ResponseWriter, r *http.Request) {
		listCommentsAPICalled++
		if r.Method != http.MethodGet {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		switch r.URL.Query().Get("page") {
		default:
			cs := []*github.PullRequestComment{
				{
					Path:     github.String("reviewdog.go"),
					Position: github.Int(1),
					Body:     github.String(serviceutil.BodyPrefix + "\nalready commented"),
				},
			}
			w.Header().Add("Link", `<https://api.github.com/repos/o/r/pulls/14/comments?page=2>; rel="next"`)
			if err := json.NewEncoder(w).Encode(cs); err != nil {
				t.Fatal(err)
			}
		case "2":
			cs := []*github.PullRequestComment{
				{
					Path:     github.String("reviewdog.go"),
					Position: github.Int(14),
					Body:     github.String(serviceutil.BodyPrefix + "\nalready commented 2"),
				},
			}
			if err := json.NewEncoder(w).Encode(cs); err != nil {
				t.Fatal(err)
			}
		}
	})
	mux.HandleFunc("/repos/o/r/pulls/14/reviews", func(w http.ResponseWriter, r *http.Request) {
		postCommentsAPICalled++
		if r.Method != http.MethodPost {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		var req github.PullRequestReviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if *req.Event != "COMMENT" {
			t.Errorf("PullRequestReviewRequest.Event = %v, want COMMENT", *req.Event)
		}
		if req.Body != nil {
			t.Errorf("PullRequestReviewRequest.Body = %v, want empty", *req.Body)
		}
		if *req.CommitID != "sha" {
			t.Errorf("PullRequestReviewRequest.Body = %v, want empty", *req.Body)
		}
		want := []*github.DraftReviewComment{
			{
				Path:     github.String("reviewdog.go"),
				Position: github.Int(14),
				Body:     github.String(serviceutil.BodyPrefix + "\nnew comment"),
			},
		}
		if diff := pretty.Compare(want, req.Comments); diff != "" {
			t.Errorf("req.Comments diff: (-got +want)\n%s", diff)
		}
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")
	g, err := NewGitHubPullRequest(cli, "o", "r", 14, "sha")
	if err != nil {
		t.Fatal(err)
	}
	comments := []*reviewdog.Comment{
		{
			CheckResult: &reviewdog.CheckResult{
				Path: "reviewdog.go",
			},
			LnumDiff: 1,
			Body:     "already commented",
		},
		{
			CheckResult: &reviewdog.CheckResult{
				Path: "reviewdog.go",
			},
			LnumDiff: 14,
			Body:     "already commented 2",
		},
		{
			CheckResult: &reviewdog.CheckResult{
				Path: "reviewdog.go",
			},
			LnumDiff: 14,
			Body:     "new comment",
		},
	}
	for _, c := range comments {
		if err := g.Post(context.Background(), c); err != nil {
			t.Error(err)
		}
	}
	if err := g.Flush(context.Background()); err != nil {
		t.Error(err)
	}
	if listCommentsAPICalled != 2 {
		t.Errorf("GitHub List PullRequest comments API called %v times, want 2 times", listCommentsAPICalled)
	}
	if postCommentsAPICalled != 1 {
		t.Errorf("GitHub post PullRequest comments API called %v times, want 1 times", postCommentsAPICalled)
	}
}

func TestGitHubPullRequest_workdir(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	moveToRootDir()

	g, err := NewGitHubPullRequest(nil, "", "", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if g.wd != "" {
		t.Fatalf("g.wd = %q, want empty", g.wd)
	}
	ctx := context.Background()
	want := "a/b/c"
	g.Post(ctx, &reviewdog.Comment{CheckResult: &reviewdog.CheckResult{Path: want}})
	if got := g.postComments[0].Path; got != want {
		t.Errorf("wd=%q path=%q, want %q", g.wd, got, want)
	}

	subDir := "cmd/"
	if err := os.Chdir(subDir); err != nil {
		t.Fatal(err)
	}
	g, _ = NewGitHubPullRequest(nil, "", "", 0, "")
	if g.wd != subDir {
		t.Fatalf("gitRelWorkdir() = %q, want %q", g.wd, subDir)
	}
	path := "a/b/c"
	wantPath := "cmd/" + path
	g.Post(ctx, &reviewdog.Comment{CheckResult: &reviewdog.CheckResult{Path: path}})
	if got := g.postComments[0].Path; got != wantPath {
		t.Errorf("wd=%q path=%q, want %q", g.wd, got, wantPath)
	}
}

func TestGitHubPullRequest_Diff_fake(t *testing.T) {
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
	g, err := NewGitHubPullRequest(cli, "o", "r", 14, "sha")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Diff(context.Background()); err != nil {
		t.Fatal(err)
	}
	if apiCalled != 1 {
		t.Errorf("GitHub API should be called once; called %v times", apiCalled)
	}
}
