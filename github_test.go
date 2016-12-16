package reviewdog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func setupGithub() (*http.ServeMux, *httptest.Server, *github.Client) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL)
	client.BaseURL = url
	client.UploadURL = url
	return mux, server, client
}

func testMethod(t *testing.T, r *http.Request, want string) {
	if got := r.Method; got != want {
		t.Errorf("Request method: %v, want %v", got, want)
	}
}

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

func TestGitHubPullRequest_Post(t *testing.T) {
	t.Skip("skipping test which post comments actually")
	client := setupGitHubClient()
	if client == nil {
		t.Skip(notokenSkipTestMes)
	}

	// https://github.com/haya14busa/reviewdog/pull/2
	owner := "haya14busa"
	repo := "reviewdog"
	pr := 2
	sha := "cce89afa9ac5519a7f5b1734db2e3aa776b138a7"

	g := NewGitHubPullReqest(client, owner, repo, pr, sha)
	comment := &Comment{
		CheckResult: &CheckResult{
			Path: "watchdogs.go",
		},
		LnumDiff: 17,
		Body:     "[reviewdog] test",
	}
	// https://github.com/haya14busa/reviewdog/pull/2/files#diff-ed1d019a10f54464cfaeaf6a736b7d27L20
	if err := g.Post(comment); err != nil {
		t.Error(err)
	}
	if err := g.Flash(); err != nil {
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
 	"github.com/haya14busa/reviewdog/diff"
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

	// https://github.com/haya14busa/reviewdog/pull/2
	owner := "haya14busa"
	repo := "reviewdog"
	pr := 2
	g := NewGitHubPullReqest(client, owner, repo, pr, "")
	b, err := g.Diff()
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
	// https://github.com/haya14busa/reviewdog/pull/2
	owner := "haya14busa"
	repo := "reviewdog"
	pr := 2
	g := NewGitHubPullReqest(client, owner, repo, pr, "")
	comments, err := g.comment()
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

func TestGitHubPullRequest_Post_Flash_mock(t *testing.T) {
	apiCalled := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != "/repos/haya14busa/reviewdog/pulls/2/comments" {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
			return
		}

		switch r.Method {
		case "GET":
		case "POST":
			var v github.PullRequestComment
			if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
				t.Error(err)
			}
			body := *v.Body
			want := `<sub>reported by [reviewdog](https://github.com/haya14busa/reviewdog) :dog:</sub>
[reviewdog] test`
			if body != want {
				t.Errorf("body: got %v, want %v", body, want)
			}
		default:
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		apiCalled++
	}))
	defer ts.Close()

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL)

	// https://github.com/haya14busa/reviewdog/pull/2
	owner := "haya14busa"
	repo := "reviewdog"
	pr := 2
	sha := "cce89afa9ac5519a7f5b1734db2e3aa776b138a7"

	g := NewGitHubPullReqest(cli, owner, repo, pr, sha)
	comment := &Comment{
		CheckResult: &CheckResult{
			Path: "reviewdog.go",
		},
		LnumDiff: 17,
		Body:     "[reviewdog] test",
	}
	// https://github.com/haya14busa/reviewdog/pull/2/files#diff-ed1d019a10f54464cfaeaf6a736b7d27L20
	if err := g.Post(comment); err != nil {
		t.Error(err)
	}
	if err := g.Flash(); err != nil {
		t.Error(err)
	}
	if apiCalled != 2 {
		t.Errorf("API should be called 2 times, but %v times", apiCalled)
	}
}

func TestGitHubPullRequest_Post_Flash_mock_review_api(t *testing.T) {
	mux, server, client := setupGithub()
	defer server.Close()
	defer func(u string) { githubAPIHost = u }(githubAPIHost)
	u, _ := url.Parse(server.URL)
	githubAPIHost = u.Host

	mux.HandleFunc("/repos/o/r/pulls/1/comments", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
	})
	mux.HandleFunc("/repos/o/r/pulls/1/reviews", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		v := new(Review)
		json.NewDecoder(r.Body).Decode(v)

		if v.Body != nil {
			t.Errorf("Review body = %v, want nil", *v.Body)
		}
		if want := "COMMENT"; *v.Event != want {
			t.Errorf("Review event = %v, want %v", *v.Event, want)
		}
		c := v.Comments[0]

		if want := bodyPrefix + "\n[reviewdog] test"; *c.Body != want {
			t.Errorf("Review comment body = %v, want %v", *c.Body, want)
		}
		if want := "reviewdog.go"; *c.Path != want {
			t.Errorf("Review comment position = %v, want %v", *c.Path, want)
		}
		if want := 17; *c.Position != want {
			t.Errorf("Review comment position = %v, want %v", *c.Position, want)
		}
	})

	g := NewGitHubPullReqest(client, "o", "r", 1, "SHA1")
	comment := &Comment{
		CheckResult: &CheckResult{
			Path: "reviewdog.go",
		},
		LnumDiff: 17,
		Body:     "[reviewdog] test",
	}
	if err := g.Post(comment); err != nil {
		t.Error(err)
	}
	if err := g.Flash(); err != nil {
		t.Error(err)
	}
}
