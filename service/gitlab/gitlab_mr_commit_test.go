package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/xanzy/go-gitlab"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
	"github.com/reviewdog/reviewdog/service/commentutil"
)

func TestGitLabMergeRequestCommitCommenter_Post_Flush_review_api(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir("../..")

	apiCalled := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/o/r/merge_requests/14/commits", func(w http.ResponseWriter, r *http.Request) {
		apiCalled++
		if r.Method != http.MethodGet {
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
		if r.Method != http.MethodGet {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		cs := []*gitlab.CommitComment{
			{
				Path: "notExistFile.go",
				Line: 1,
				Note: commentutil.BodyPrefix + "already commented",
			},
		}
		if err := json.NewEncoder(w).Encode(cs); err != nil {
			t.Fatal(err)
		}
	})
	mux.HandleFunc("/api/v4/projects/o/r/repository/commits/sha/comments", func(w http.ResponseWriter, r *http.Request) {
		apiCalled++
		if r.Method != http.MethodPost {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		var req gitlab.CommitComment
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		want := gitlab.CommitComment{
			Path:     "notExistFile.go",
			Line:     14,
			Note:     commentutil.BodyPrefix + "new comment",
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

	cli, err := gitlab.NewClient("", gitlab.WithBaseURL(ts.URL+"/api/v4"))
	if err != nil {
		t.Fatal(err)
	}

	g, err := NewGitLabMergeRequestCommitCommenter(cli, "o", "r", 14, "sha")
	if err != nil {
		t.Fatal(err)
	}
	// Path is set to non existing file path for mock test not to use last commit id of the line.
	// If setting exists file path, sha is changed by last commit id.
	comments := []*reviewdog.Comment{
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "notExistFile.go",
						Range: &rdf.Range{Start: &rdf.Position{
							Line: 1,
						}},
					},
					Message: "already commented",
				},
				InDiffFile: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "notExistFile.go",
						Range: &rdf.Range{Start: &rdf.Position{
							Line: 14,
						}},
					},
					Message: "new comment",
				},
				InDiffFile: true,
			},
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

func TestGitLabPullRequest_workdir(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir("../..")

	g, err := NewGitLabMergeRequestCommitCommenter(nil, "", "", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if g.wd != "" {
		t.Fatalf("g.wd = %q, want empty", g.wd)
	}
	ctx := context.Background()
	want := "a/b/c"
	g.Post(ctx, &reviewdog.Comment{Result: &filter.FilteredDiagnostic{
		Diagnostic: &rdf.Diagnostic{Location: &rdf.Location{Path: want}}}})
	if got := g.postComments[0].Result.Diagnostic.GetLocation().GetPath(); got != want {
		t.Errorf("wd=%q path=%q, want %q", g.wd, got, want)
	}

	subDir := "cmd/"
	if err := os.Chdir(subDir); err != nil {
		t.Fatal(err)
	}
	g, _ = NewGitLabMergeRequestCommitCommenter(nil, "", "", 0, "")
	if g.wd != subDir {
		t.Fatalf("gitRelWorkdir() = %q, want %q", g.wd, subDir)
	}
	path := "a/b/c"
	wantPath := "cmd/" + path
	g.Post(ctx, &reviewdog.Comment{Result: &filter.FilteredDiagnostic{
		Diagnostic: &rdf.Diagnostic{Location: &rdf.Location{Path: want}}}})
	if got := g.postComments[0].Result.Diagnostic.GetLocation().GetPath(); got != wantPath {
		t.Errorf("wd=%q path=%q, want %q", g.wd, got, wantPath)
	}
}
