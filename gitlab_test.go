package reviewdog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/xanzy/go-gitlab"
)

func TestGitLabPullRequest_Post_Flush_review_api(t *testing.T) {
	apiCalled := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/o/r/merge_requests/14/commits", func(w http.ResponseWriter, r *http.Request) {
		apiCalled++
		if r.Method != "GET" {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		cs := []*gitlab.Commit{
			{
				ID:      "0123456789abcdef",
				ShortID: "012345678",
			},
		}
		if err := json.NewEncoder(w).Encode(cs); err != nil {
			t.Fatal(err)
		}
	})
	mux.HandleFunc("/api/v4/projects/o/r/repository/commits/0123456789abcdef/comments", func(w http.ResponseWriter, r *http.Request) {
		apiCalled++
		if r.Method != "GET" {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		cs := []*gitlab.CommitComment{
			{
				Path: "notExistFile.go",
				Line: 1,
				Note: bodyPrefix + "\nalready commented",
			},
		}
		if err := json.NewEncoder(w).Encode(cs); err != nil {
			t.Fatal(err)
		}
	})
	mux.HandleFunc("/api/v4/projects/o/r/repository/commits/sha/comments", func(w http.ResponseWriter, r *http.Request) {
		apiCalled++
		if r.Method != "POST" {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		var req gitlab.CommitComment
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		want := gitlab.CommitComment{
			Path:     "notExistFile.go",
			Line:     14,
			Note:     bodyPrefix + "\nnew comment",
			LineType: "new",
		}
		if diff := pretty.Compare(want, req); diff != "" {
			t.Errorf("req.Comments diff: (-got +want)\n%s", diff)
		}
		if err := json.NewEncoder(w).Encode(req); err != nil {
			t.Fatal(err)
		}
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := gitlab.NewClient(nil, "")
	cli.SetBaseURL(ts.URL)
	g, err := NewGitLabMergeReqest(cli, "o", "r", 14, "sha")
	if err != nil {
		t.Fatal(err)
	}
	// Path is set to non existing file path for mock test not to use last commit id of the line.
	// If setting exists file path, sha is changed by last commit id.
	comments := []*Comment{
		{
			CheckResult: &CheckResult{
				Path: "notExistFile.go",
				Lnum: 1,
			},
			Body: "already commented",
		},
		{
			CheckResult: &CheckResult{
				Path: "notExistFile.go",
				Lnum: 14,
			},
			Body: "new comment",
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
	if want := 3; apiCalled != want {
		t.Errorf("GitLab API is called %d times, want %d times", apiCalled, want)
	}
}

func TestGitLabPullReqest_workdir(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)

	g, err := NewGitLabMergeReqest(nil, "", "", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if g.wd != "" {
		t.Fatalf("g.wd = %q, want empty", g.wd)
	}
	ctx := context.Background()
	want := "a/b/c"
	g.Post(ctx, &Comment{CheckResult: &CheckResult{Path: want}})
	if got := g.postComments[0].Path; got != want {
		t.Errorf("wd=%q path=%q, want %q", g.wd, got, want)
	}

	subDir := "cmd/"
	if err := os.Chdir(subDir); err != nil {
		t.Fatal(err)
	}
	g, _ = NewGitLabMergeReqest(nil, "", "", 0, "")
	if g.wd != subDir {
		t.Fatalf("gitRelWorkdir() = %q, want %q", g.wd, subDir)
	}
	path := "a/b/c"
	wantPath := "cmd/" + path
	g.Post(ctx, &Comment{CheckResult: &CheckResult{Path: path}})
	if got := g.postComments[0].Path; got != wantPath {
		t.Errorf("wd=%q path=%q, want %q", g.wd, got, wantPath)
	}
}
