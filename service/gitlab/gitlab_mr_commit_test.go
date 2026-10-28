package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	gitlab "gitlab.com/gitlab-org/api/client-go/v3"

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
	mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14/commits", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("/api/v4/projects/o%2Fr/repository/commits/0123456789abcdef/comments", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("/api/v4/projects/o%2Fr/repository/commits/sha/comments", func(w http.ResponseWriter, r *http.Request) {
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

	cli, err := gitlab.NewClient("", gitlab.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatal(err)
	}

	g := NewGitLabMergeRequestCommitCommenter(cli, "o", "r", 14, "sha")

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

func TestGitLabMergeRequestCommitCommenter_Flush_lastCommitOfLine(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	git := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	commit := func(dir, content, msg string) {
		t.Helper()
		if err := os.WriteFile(dir+"/a.txt", []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		git(dir, "add", "a.txt")
		git(dir, "commit", "-q", "-m", msg)
	}

	src := t.TempDir()
	git(src, "init", "-q")
	git(src, "config", "user.name", "reviewdog")
	git(src, "config", "user.email", "reviewdog@example.com")
	commit(src, "a\nb\nc\n", "first")
	commit(src, "a\nb\nc\nd\n", "second")
	commit(src, "x\ny\na\nb\nc\nd\n", "third")
	second := git(src, "rev-parse", "HEAD^")
	head := git(src, "rev-parse", "HEAD")

	// In a shallow clone, git blame marks lines from before the clone's
	// oldest commit with a "^" in front of the commit ID.
	shallow := t.TempDir()
	git(shallow, "clone", "-q", "--depth", "1", "file://"+src, ".")
	// Lines that aren't committed have no commit to comment on.
	if err := os.WriteFile(shallow+"/a.txt", []byte("x\ny\na\nb\nc\nd\nz\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		dir        string
		line       int32
		wantCommit string
		wantLine   int64
	}{
		{
			name:       "line moved by a later commit",
			dir:        src,
			line:       6,
			wantCommit: second,
			wantLine:   4,
		},
		{
			name:       "boundary commit of a shallow clone",
			dir:        shallow,
			line:       3,
			wantCommit: head,
			wantLine:   3,
		},
		{
			name:       "line not committed yet",
			dir:        shallow,
			line:       7,
			wantCommit: head,
			wantLine:   7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(tt.dir)

			posted := 0
			mux := http.NewServeMux()
			mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14/commits", func(w http.ResponseWriter, _ *http.Request) {
				if err := json.NewEncoder(w).Encode([]*gitlab.Commit{}); err != nil {
					t.Fatal(err)
				}
			})
			mux.HandleFunc("/api/v4/projects/o%2Fr/repository/commits/"+tt.wantCommit+"/comments", func(w http.ResponseWriter, r *http.Request) {
				posted++
				var req gitlab.CommitComment
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
				}
				if req.Path != "a.txt" || req.Line != tt.wantLine {
					t.Errorf("comment posted at %s:%d, want a.txt:%d", req.Path, req.Line, tt.wantLine)
				}
				if err := json.NewEncoder(w).Encode(req); err != nil {
					t.Fatal(err)
				}
			})
			ts := httptest.NewServer(mux)
			defer ts.Close()

			cli, err := gitlab.NewClient("", gitlab.WithBaseURL(ts.URL))
			if err != nil {
				t.Fatal(err)
			}
			g := NewGitLabMergeRequestCommitCommenter(cli, "o", "r", 14, head)
			c := &reviewdog.Comment{
				Result: &filter.FilteredDiagnostic{
					Diagnostic: &rdf.Diagnostic{
						Location: &rdf.Location{
							Path:  "a.txt",
							Range: &rdf.Range{Start: &rdf.Position{Line: tt.line}},
						},
						Message: "comment",
					},
					InDiffFile: true,
				},
			}
			if err := g.Post(context.Background(), c); err != nil {
				t.Fatal(err)
			}
			if err := g.Flush(context.Background()); err != nil {
				t.Fatal(err)
			}
			if posted != 1 {
				t.Errorf("comment posted to commit %s %d times, want 1", tt.wantCommit, posted)
			}
		})
	}
}
