package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
				ID:      "sha",
				ShortID: "sha",
			},
		}
		if err := json.NewEncoder(w).Encode(cs); err != nil {
			t.Fatal(err)
		}
	})
	mux.HandleFunc("GET /api/v4/projects/o%2Fr/repository/commits/sha/comments", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("POST /api/v4/projects/o%2Fr/repository/commits/sha/comments", func(w http.ResponseWriter, r *http.Request) {
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
	write := func(dir, name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	commit := func(dir, msg string) {
		t.Helper()
		git(dir, "add", "-A")
		git(dir, "commit", "-q", "-m", msg)
	}

	src := t.TempDir()
	git(src, "init", "-q")
	git(src, "config", "user.name", "reviewdog")
	git(src, "config", "user.email", "reviewdog@example.com")
	write(src, "a.txt", "a\nb\nc\n")
	commit(src, "first")
	write(src, "a.txt", "a\nb\nc\nd\n")
	// git quotes non-ASCII paths in its output.
	write(src, "旧.txt", "1\n2\n")
	commit(src, "second")
	write(src, "a.txt", "x\ny\na\nb\nc\nd\n")
	git(src, "mv", "旧.txt", "new.txt")
	commit(src, "third")
	second := git(src, "rev-parse", "HEAD^")
	head := git(src, "rev-parse", "HEAD")

	// In a shallow clone, git blame stops at the clone's oldest commit, and
	// marks it with a "^" in front of the commit ID in its default format.
	shallow := t.TempDir()
	git(shallow, "clone", "-q", "--depth", "2", "file://"+src, ".")
	// Lines that aren't committed have no commit to comment on.
	write(shallow, "a.txt", "x\ny\na\nb\nc\nd\nz\n")

	type position struct {
		commit string
		path   string
		line   int64
	}
	tests := []struct {
		name string
		dir  string
		path string
		line int32
		// mrCommits are the merge request's commits. Defaults to second and
		// head, as if first were on the target branch.
		mrCommits []string
		// existing is where the same comment has already been posted.
		existing *position
		// want is where the comment is posted, or nil if it isn't.
		want *position
	}{
		{
			name: "line moved by a later commit",
			dir:  src,
			path: "a.txt",
			line: 6,
			want: &position{second, "a.txt", 4},
		},
		{
			name: "file renamed by a later commit",
			dir:  src,
			path: "new.txt",
			line: 2,
			want: &position{second, "旧.txt", 2},
		},
		{
			name: "line from a commit outside the merge request",
			dir:  src,
			path: "a.txt",
			line: 3,
			want: &position{head, "a.txt", 3},
		},
		{
			name: "boundary commit of a shallow clone",
			dir:  shallow,
			path: "a.txt",
			line: 3,
			want: &position{second, "a.txt", 1},
		},
		{
			name: "line not committed yet",
			dir:  shallow,
			path: "a.txt",
			line: 7,
			want: &position{head, "a.txt", 7},
		},
		{
			name:     "already posted on the blamed line",
			dir:      src,
			path:     "a.txt",
			line:     6,
			existing: &position{second, "a.txt", 4},
		},
		{
			name:     "already posted on the blamed path",
			dir:      src,
			path:     "new.txt",
			line:     2,
			existing: &position{second, "旧.txt", 2},
		},
		{
			name:     "same line posted on another commit",
			dir:      src,
			path:     "a.txt",
			line:     6,
			existing: &position{head, "a.txt", 4},
			want:     &position{second, "a.txt", 4},
		},
		{
			name:     "already posted on the head commit",
			dir:      src,
			path:     "a.txt",
			line:     3,
			existing: &position{head, "a.txt", 3},
		},
		{
			// In merged results pipelines, the head commit is a merge commit
			// that isn't part of the merge request.
			name:      "already posted on a head commit outside the merge request",
			dir:       src,
			path:      "a.txt",
			line:      3,
			mrCommits: []string{second},
			existing:  &position{head, "a.txt", 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(tt.dir)

			c := &reviewdog.Comment{
				Result: &filter.FilteredDiagnostic{
					Diagnostic: &rdf.Diagnostic{
						Location: &rdf.Location{
							Path:  tt.path,
							Range: &rdf.Range{Start: &rdf.Position{Line: tt.line}},
						},
						Message: "comment",
					},
					InDiffFile: true,
				},
			}
			body := commentutil.MarkdownComment(c)

			var posted []position
			mux := http.NewServeMux()
			mux.HandleFunc("GET /api/v4/projects/o%2Fr/merge_requests/14/commits", func(w http.ResponseWriter, _ *http.Request) {
				mrCommits := tt.mrCommits
				if mrCommits == nil {
					mrCommits = []string{head, second}
				}
				var cs []*gitlab.Commit
				for _, id := range mrCommits {
					cs = append(cs, &gitlab.Commit{ID: id})
				}
				if err := json.NewEncoder(w).Encode(cs); err != nil {
					t.Error(err)
					return
				}
			})
			mux.HandleFunc("GET /api/v4/projects/o%2Fr/repository/commits/{sha}/comments", func(w http.ResponseWriter, r *http.Request) {
				cs := []*gitlab.CommitComment{}
				if e := tt.existing; e != nil && e.commit == r.PathValue("sha") {
					cs = append(cs, &gitlab.CommitComment{Path: e.path, Line: e.line, Note: body})
				}
				if err := json.NewEncoder(w).Encode(cs); err != nil {
					t.Error(err)
					return
				}
			})
			mux.HandleFunc("POST /api/v4/projects/o%2Fr/repository/commits/{sha}/comments", func(w http.ResponseWriter, r *http.Request) {
				var req gitlab.CommitComment
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					return
				}
				posted = append(posted, position{r.PathValue("sha"), req.Path, req.Line})
				if err := json.NewEncoder(w).Encode(req); err != nil {
					t.Error(err)
					return
				}
			})
			ts := httptest.NewServer(mux)
			defer ts.Close()

			cli, err := gitlab.NewClient("", gitlab.WithBaseURL(ts.URL))
			if err != nil {
				t.Fatal(err)
			}
			g := NewGitLabMergeRequestCommitCommenter(cli, "o", "r", 14, head)
			if err := g.Post(context.Background(), c); err != nil {
				t.Fatal(err)
			}
			if err := g.Flush(context.Background()); err != nil {
				t.Fatal(err)
			}
			var want []position
			if tt.want != nil {
				want = append(want, *tt.want)
			}
			if !slices.Equal(posted, want) {
				t.Errorf("comments posted at %v, want %v", posted, want)
			}
		})
	}
}
