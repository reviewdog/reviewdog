package watchdogs

import (
	"os"
	"testing"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const notokenSkipTestMes = "skipping test (requires actual Personal access tokens. export WATCHDOGS_TEST_GITHUB_API_TOKEN=<GitHub Personal Access Token>)"

func setupGitHubClient() *github.Client {
	token := os.Getenv("WATCHDOGS_TEST_GITHUB_API_TOKEN")
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

	// https://github.com/haya14busa/watchdogs/pull/2
	owner := "haya14busa"
	repo := "watchdogs"
	pr := 2
	sha := "cce89afa9ac5519a7f5b1734db2e3aa776b138a7"

	g := NewGitHubPullReqest(client, owner, repo, pr, sha)
	comment := &Comment{
		CheckResult: &CheckResult{
			Path: "watchdogs.go",
		},
		LnumDiff: 17,
		Body:     "[watchdogs] test",
	}
	// https://github.com/haya14busa/watchdogs/pull/2/files#diff-ed1d019a10f54464cfaeaf6a736b7d27L20
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
diff --git a/watchdogs.go b/watchdogs.go
index 61450f3..f63f149 100644
--- a/watchdogs.go
+++ b/watchdogs.go
@@ -10,18 +10,18 @@ import (
 	"github.com/haya14busa/watchdogs/diff"
 )
 
+var TestExportedVarWithoutComment = 1
+
+func NewWatchdogs(p Parser, c CommentService, d DiffService) *Watchdogs {
+	return &Watchdogs{p: p, c: c, d: d}
+}
+
 type Watchdogs struct {
 	p Parser
 	c CommentService
 	d DiffService
 }
 
-func NewWatchdogs(p Parser, c CommentService, d DiffService) *Watchdogs {
-	return &Watchdogs{p: p, c: c, d: d}
-}
-
-// CheckResult represents a checked result of static analysis tools.
-// :h error-file-format
 type CheckResult struct {
 	Path    string   // file path
 	Lnum    int      // line number
`

	// https://github.com/haya14busa/watchdogs/pull/2
	owner := "haya14busa"
	repo := "watchdogs"
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
	// https://github.com/haya14busa/watchdogs/pull/2
	owner := "haya14busa"
	repo := "watchdogs"
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
