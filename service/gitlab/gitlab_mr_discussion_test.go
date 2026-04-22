package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
	"github.com/reviewdog/reviewdog/service/commentutil"
	"github.com/reviewdog/reviewdog/service/serviceutil"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
)

// metaBody returns the body that reviewdog would post for the given comment.
// It matches the format produced by postCommentsForEach.
func metaBody(t *testing.T, c *reviewdog.Comment, toolName string) string {
	t.Helper()
	fprint, err := serviceutil.Fingerprint(c.Result.Diagnostic)
	if err != nil {
		t.Fatal(err)
	}
	body := commentutil.MarkdownComment(c)
	if suggestion := buildSuggestions(c); suggestion != "" {
		body += "\n\n" + suggestion
	}
	body += "\n" + serviceutil.BuildMetaComment(fprint, toolName) + "\n"
	return body
}

func metaBodyForNote(t *testing.T, c *reviewdog.Comment, toolName string) string {
	t.Helper()
	fprint, err := serviceutil.Fingerprint(c.Result.Diagnostic)
	if err != nil {
		t.Fatal(err)
	}
	return commentutil.MarkdownComment(c) + "\n" + serviceutil.BuildMetaComment(fprint, toolName) + "\n"
}

func TestGitLabMergeRequestDiscussionCommenter_Post_Flush_review_api(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir("../..")

	alreadyCommented1 := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: "file.go",
					Range: &rdf.Range{Start: &rdf.Position{
						Line: 1,
					}},
				},
				Message: "already commented",
			},
			InDiffFile: true,
		},
	}
	alreadyCommented2 := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: "another/file.go",
					Range: &rdf.Range{Start: &rdf.Position{
						Line: 14,
					}},
				},
				Message: "already commented 2",
			},
			InDiffFile: true,
		},
	}
	newComment1 := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: "file.go",
					Range: &rdf.Range{Start: &rdf.Position{
						Line: 14,
					}},
				},
				Message: "new comment",
			},
			InDiffFile: true,
		},
	}
	newComment2 := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: "file2.go",
					Range: &rdf.Range{Start: &rdf.Position{
						Line: 15,
					}},
				},
				Message: "new comment 2",
			},
			InDiffFile: true,
		},
	}
	newComment3 := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: "new_file.go",
					Range: &rdf.Range{Start: &rdf.Position{
						Line: 14,
					}},
				},
				Message: "new comment 3",
			},
			OldPath:    "old_file.go",
			OldLine:    7,
			InDiffFile: true,
		},
	}
	commentOutsideDiff := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: "path.go",
					Range: &rdf.Range{Start: &rdf.Position{
						Line: 14,
					}},
				},
				Message: "comment outside diff",
			},
			InDiffFile: false,
		},
	}
	commentWithoutLnum := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: "path.go",
				},
				Message: "comment without lnum",
			},
			InDiffFile: true,
		},
	}

	newCommentWithSuggestion := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: "file3.go",
					Range: &rdf.Range{Start: &rdf.Position{
						Line: 14,
					}},
				},
				Message: "new comment with suggestion",
				Suggestions: []*rdf.Suggestion{
					{
						Text: "line1-fixed\nline2-fixed",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 14,
							},
							End: &rdf.Position{
								Line: 15,
							},
						},
					},
				},
			},
			InDiffFile: true,
		},
	}

	comments := []*reviewdog.Comment{
		alreadyCommented1,
		alreadyCommented2,
		newComment1,
		newComment2,
		newComment3,
		commentOutsideDiff,
		commentWithoutLnum,
		newCommentWithSuggestion,
	}
	var postCalled int32
	const wantPostCalled = 4

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14/discussions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			switch r.URL.Query().Get("page") {
			default:
				dls := []*gitlab.Discussion{
					{
						ID: "already-1",
						Notes: []*gitlab.Note{
							{
								Body:       metaBodyForNote(t, alreadyCommented1, "tool-name"),
								Resolvable: true,
								Position: &gitlab.NotePosition{
									NewPath: alreadyCommented1.Result.Diagnostic.GetLocation().GetPath(),
									NewLine: int64(alreadyCommented1.Result.Diagnostic.GetLocation().GetRange().GetStart().GetLine()),
								},
							},
							{
								Body: "unrelated commented",
								Position: &gitlab.NotePosition{
									NewPath: "file.go",
									NewLine: 1,
								},
							},
						},
					},
				}
				w.Header().Add("X-Next-Page", "2")
				if err := json.NewEncoder(w).Encode(dls); err != nil {
					t.Fatal(err)
				}
			case "2":
				dls := []*gitlab.Discussion{
					{
						ID: "already-2",
						Notes: []*gitlab.Note{
							{
								Body:       metaBodyForNote(t, alreadyCommented2, "tool-name"),
								Resolvable: true,
								Position: &gitlab.NotePosition{
									NewPath: alreadyCommented2.Result.Diagnostic.GetLocation().GetPath(),
									NewLine: int64(alreadyCommented2.Result.Diagnostic.GetLocation().GetRange().GetStart().GetLine()),
								},
							},
						},
					},
				}
				if err := json.NewEncoder(w).Encode(dls); err != nil {
					t.Fatal(err)
				}
			}

		case http.MethodPost:
			atomic.AddInt32(&postCalled, 1)
			got := new(gitlab.CreateMergeRequestDiscussionOptions)
			if err := json.NewDecoder(r.Body).Decode(got); err != nil {
				t.Error(err)
			}
			switch *got.Position.NewPath {
			case "file.go":
				want := &gitlab.CreateMergeRequestDiscussionOptions{
					Body: gitlab.Ptr(metaBody(t, newComment1, "tool-name")),
					Position: &gitlab.PositionOptions{
						BaseSHA:      gitlab.Ptr("xxx"),
						StartSHA:     gitlab.Ptr("xxx"),
						HeadSHA:      gitlab.Ptr("sha"),
						PositionType: gitlab.Ptr("text"),
						NewPath:      gitlab.Ptr("file.go"),
						NewLine:      gitlab.Ptr(int64(14)),
					},
				}
				if diff := cmp.Diff(got, want); diff != "" {
					t.Error(diff)
				}
			case "file2.go":
				want := &gitlab.CreateMergeRequestDiscussionOptions{
					Body: gitlab.Ptr(metaBody(t, newComment2, "tool-name")),
					Position: &gitlab.PositionOptions{
						BaseSHA:      gitlab.Ptr("xxx"),
						StartSHA:     gitlab.Ptr("xxx"),
						HeadSHA:      gitlab.Ptr("sha"),
						PositionType: gitlab.Ptr("text"),
						NewPath:      gitlab.Ptr("file2.go"),
						NewLine:      gitlab.Ptr(int64(15)),
					},
				}
				if diff := cmp.Diff(got, want); diff != "" {
					t.Error(diff)
				}
			case "new_file.go":
				want := &gitlab.CreateMergeRequestDiscussionOptions{
					Body: gitlab.Ptr(metaBody(t, newComment3, "tool-name")),
					Position: &gitlab.PositionOptions{
						BaseSHA:      gitlab.Ptr("xxx"),
						StartSHA:     gitlab.Ptr("xxx"),
						HeadSHA:      gitlab.Ptr("sha"),
						PositionType: gitlab.Ptr("text"),
						NewPath:      gitlab.Ptr("new_file.go"),
						NewLine:      gitlab.Ptr(int64(14)),
						OldPath:      gitlab.Ptr("old_file.go"),
						OldLine:      gitlab.Ptr(int64(7)),
					},
				}
				if diff := cmp.Diff(got, want); diff != "" {
					t.Error(diff)
				}
			case "file3.go":
				want := &gitlab.CreateMergeRequestDiscussionOptions{
					Body: gitlab.Ptr(metaBody(t, newCommentWithSuggestion, "tool-name")),
					Position: &gitlab.PositionOptions{
						BaseSHA:      gitlab.Ptr("xxx"),
						StartSHA:     gitlab.Ptr("xxx"),
						HeadSHA:      gitlab.Ptr("sha"),
						PositionType: gitlab.Ptr("text"),
						NewPath:      gitlab.Ptr("file3.go"),
						NewLine:      gitlab.Ptr(int64(14)),
					},
				}
				if diff := cmp.Diff(got, want); diff != "" {
					t.Error(diff)
				}
			default:
				t.Errorf("got unexpected discussion: %#v", got)
			}
			if err := json.NewEncoder(w).Encode(gitlab.Discussion{}); err != nil {
				t.Fatal(err)
			}
		default:
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
	})
	mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		w.Write([]byte(`{"target_project_id": 14, "target_branch": "test-branch"}`))
	})
	mux.HandleFunc("/api/v4/projects/14/repository/branches/test-branch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		w.Write([]byte(`{"commit": {"id": "xxx"}}`))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli, err := gitlab.NewClient("", gitlab.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatal(err)
	}

	g := NewGitLabMergeRequestDiscussionCommenter(cli, "o", "r", 14, "sha", "tool-name")

	for _, c := range comments {
		if err := g.Post(context.Background(), c); err != nil {
			t.Error(err)
		}
	}
	if err := g.Flush(context.Background()); err != nil {
		t.Errorf("%v", err)
	}
	if postCalled != wantPostCalled {
		t.Errorf("%d discussions posted, but want %d", postCalled, wantPostCalled)
	}
}

func TestGitLabMergeRequestDiscussionCommenter_Flush_resolvesOutdatedDiscussions(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir("../..")

	// Diagnostic that WILL be reported again this run — must not be resolved.
	stillReported := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path:  "file.go",
					Range: &rdf.Range{Start: &rdf.Position{Line: 1}},
				},
				Message: "still reported",
			},
			InDiffFile: true,
		},
	}
	// Diagnostic that was previously posted but is NOT reported this run —
	// its discussion should be resolved.
	fixed := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path:  "file.go",
					Range: &rdf.Range{Start: &rdf.Position{Line: 2}},
				},
				Message: "fixed in new run",
			},
			InDiffFile: true,
		},
	}
	// A previously-posted comment from a DIFFERENT tool must not be resolved
	// even if not reported this run.
	otherTool := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path:  "file.go",
					Range: &rdf.Range{Start: &rdf.Position{Line: 3}},
				},
				Message: "from another tool",
			},
			InDiffFile: true,
		},
	}

	var resolved sync.Map

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14/discussions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			// Accept the POST for stillReported? It won't POST since it's already posted.
			// If any unexpected POST happens, record it.
			if r.Method == http.MethodPost {
				t.Errorf("unexpected discussion create: %v", r.URL)
				return
			}
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
			return
		}
		dls := []*gitlab.Discussion{
			{
				ID: "disc-still-reported",
				Notes: []*gitlab.Note{{
					Body:       metaBodyForNote(t, stillReported, "tool-name"),
					Resolvable: true,
					Position: &gitlab.NotePosition{
						NewPath: "file.go",
						NewLine: 1,
					},
				}},
			},
			{
				ID: "disc-fixed",
				Notes: []*gitlab.Note{{
					Body:       metaBodyForNote(t, fixed, "tool-name"),
					Resolvable: true,
					Position: &gitlab.NotePosition{
						NewPath: "file.go",
						NewLine: 2,
					},
				}},
			},
			{
				ID: "disc-already-resolved",
				Notes: []*gitlab.Note{{
					Body:       metaBodyForNote(t, fixed, "tool-name"),
					Resolvable: true,
					Resolved:   true,
					Position: &gitlab.NotePosition{
						NewPath: "file.go",
						NewLine: 9,
					},
				}},
			},
			{
				ID: "disc-other-tool",
				Notes: []*gitlab.Note{{
					Body:       metaBodyForNote(t, otherTool, "different-tool"),
					Resolvable: true,
					Position: &gitlab.NotePosition{
						NewPath: "file.go",
						NewLine: 3,
					},
				}},
			},
		}
		if err := json.NewEncoder(w).Encode(dls); err != nil {
			t.Fatal(err)
		}
	})
	// Resolve endpoint: PUT /projects/:id/merge_requests/:mr/discussions/:discussion_id
	mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14/discussions/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method on resolve: %v %v", r.Method, r.URL)
			return
		}
		// URL form: .../discussions/<id>
		parts := strings.Split(r.URL.Path, "/")
		id := parts[len(parts)-1]
		if r.URL.Query().Get("resolved") != "true" {
			// client-go sends as JSON body, not query; accept either.
			var opts gitlab.ResolveMergeRequestDiscussionOptions
			_ = json.NewDecoder(r.Body).Decode(&opts)
			if opts.Resolved == nil || !*opts.Resolved {
				t.Errorf("expected resolve=true, got %+v", opts)
			}
		}
		resolved.Store(id, true)
		if err := json.NewEncoder(w).Encode(gitlab.Discussion{ID: id}); err != nil {
			t.Fatal(err)
		}
	})
	mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"target_project_id": 14, "target_branch": "test-branch"}`))
	})
	mux.HandleFunc("/api/v4/projects/14/repository/branches/test-branch", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"commit": {"id": "xxx"}}`))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli, err := gitlab.NewClient("", gitlab.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatal(err)
	}

	g := NewGitLabMergeRequestDiscussionCommenter(cli, "o", "r", 14, "sha", "tool-name")
	// Only report the stillReported diagnostic — fixed/otherTool are absent this run.
	if err := g.Post(context.Background(), stillReported); err != nil {
		t.Fatal(err)
	}
	if err := g.Flush(context.Background()); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	if _, ok := resolved.Load("disc-fixed"); !ok {
		t.Errorf("expected disc-fixed to be resolved")
	}
	if _, ok := resolved.Load("disc-still-reported"); ok {
		t.Errorf("disc-still-reported must NOT be resolved (diagnostic is still reported)")
	}
	if _, ok := resolved.Load("disc-other-tool"); ok {
		t.Errorf("disc-other-tool must NOT be resolved (different tool)")
	}
	if _, ok := resolved.Load("disc-already-resolved"); ok {
		t.Errorf("disc-already-resolved must NOT be re-resolved")
	}
}

// TestGitLabMergeRequestDiscussionCommenter_Flush_skipsUnresolvableAndLegacy
// verifies that discussions without a resolvable note (GitLab returns
// Resolvable=false for some thread kinds) and legacy reviewdog notes without
// an embedded meta comment are never passed to the resolve endpoint, even when
// the diagnostic is no longer reported this run.
func TestGitLabMergeRequestDiscussionCommenter_Flush_skipsUnresolvableAndLegacy(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir("../..")

	legacyComment := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path:  "file.go",
					Range: &rdf.Range{Start: &rdf.Position{Line: 5}},
				},
				Message: "legacy note without meta",
			},
			InDiffFile: true,
		},
	}
	unresolvable := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path:  "file.go",
					Range: &rdf.Range{Start: &rdf.Position{Line: 6}},
				},
				Message: "unresolvable thread",
			},
			InDiffFile: true,
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14/discussions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			dls := []*gitlab.Discussion{
				{
					ID: "disc-legacy",
					Notes: []*gitlab.Note{{
						Body:       commentutil.MarkdownComment(legacyComment),
						Resolvable: true,
						Position: &gitlab.NotePosition{
							NewPath: "file.go",
							NewLine: 5,
						},
					}},
				},
				{
					ID: "disc-unresolvable",
					Notes: []*gitlab.Note{{
						Body:       metaBodyForNote(t, unresolvable, "tool-name"),
						Resolvable: false,
						Position: &gitlab.NotePosition{
							NewPath: "file.go",
							NewLine: 6,
						},
					}},
				},
			}
			if err := json.NewEncoder(w).Encode(dls); err != nil {
				t.Fatal(err)
			}
		case http.MethodPost:
			// Legacy comment must be recognized as already-posted.
			got := new(gitlab.CreateMergeRequestDiscussionOptions)
			_ = json.NewDecoder(r.Body).Decode(got)
			t.Errorf("unexpected discussion POST: %#v", got)
		default:
			t.Errorf("unexpected method: %v %v", r.Method, r.URL)
		}
	})
	mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14/discussions/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected resolve call: %v %v", r.Method, r.URL)
	})
	mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"target_project_id": 14, "target_branch": "test-branch"}`))
	})
	mux.HandleFunc("/api/v4/projects/14/repository/branches/test-branch", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"commit": {"id": "xxx"}}`))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli, err := gitlab.NewClient("", gitlab.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatal(err)
	}

	g := NewGitLabMergeRequestDiscussionCommenter(cli, "o", "r", 14, "sha", "tool-name")
	// Re-report the legacy diagnostic so it is matched as already-posted.
	// Do NOT re-report `unresolvable` — this would normally trigger a resolve,
	// but Resolvable=false must prevent that.
	if err := g.Post(context.Background(), legacyComment); err != nil {
		t.Fatal(err)
	}
	if err := g.Flush(context.Background()); err != nil {
		t.Fatalf("Flush: %v", err)
	}
}

// TestGitLabMergeRequestDiscussionCommenter_Flush_propagatesResolveError
// ensures a failing resolve surfaces as a Flush error rather than being
// silently swallowed.
func TestGitLabMergeRequestDiscussionCommenter_Flush_propagatesResolveError(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir("../..")

	fixed := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path:  "file.go",
					Range: &rdf.Range{Start: &rdf.Position{Line: 2}},
				},
				Message: "fixed in new run",
			},
			InDiffFile: true,
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14/discussions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %v", r.Method)
			return
		}
		dls := []*gitlab.Discussion{{
			ID: "disc-fixed",
			Notes: []*gitlab.Note{{
				Body:       metaBodyForNote(t, fixed, "tool-name"),
				Resolvable: true,
				Position: &gitlab.NotePosition{
					NewPath: "file.go",
					NewLine: 2,
				},
			}},
		}}
		_ = json.NewEncoder(w).Encode(dls)
	})
	mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14/discussions/disc-fixed", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"boom"}`, http.StatusInternalServerError)
	})
	mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"target_project_id": 14, "target_branch": "test-branch"}`))
	})
	mux.HandleFunc("/api/v4/projects/14/repository/branches/test-branch", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"commit": {"id": "xxx"}}`))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli, err := gitlab.NewClient("", gitlab.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatal(err)
	}

	g := NewGitLabMergeRequestDiscussionCommenter(cli, "o", "r", 14, "sha", "tool-name")
	// Do not re-post `fixed` — it becomes outdated and triggers a resolve,
	// which the mock server rejects with 500.
	err = g.Flush(context.Background())
	if err == nil {
		t.Fatalf("expected Flush error, got nil")
	}
	if !strings.Contains(err.Error(), "disc-fixed") {
		t.Errorf("expected error to reference discussion id, got: %v", err)
	}
}

// TestGitLabMergeRequestDiscussionCommenter_Flush_paginatedOutdated confirms
// that outdated discussions appearing on page 2+ of the discussion listing are
// still picked up for auto-resolve.
func TestGitLabMergeRequestDiscussionCommenter_Flush_paginatedOutdated(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir("../..")

	fixed := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path:  "file.go",
					Range: &rdf.Range{Start: &rdf.Position{Line: 7}},
				},
				Message: "fixed on page 2",
			},
			InDiffFile: true,
		},
	}

	var resolved sync.Map

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14/discussions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %v", r.Method)
			return
		}
		switch r.URL.Query().Get("page") {
		default:
			w.Header().Add("X-Next-Page", "2")
			_ = json.NewEncoder(w).Encode([]*gitlab.Discussion{})
		case "2":
			dls := []*gitlab.Discussion{{
				ID: "disc-page2-fixed",
				Notes: []*gitlab.Note{{
					Body:       metaBodyForNote(t, fixed, "tool-name"),
					Resolvable: true,
					Position: &gitlab.NotePosition{
						NewPath: "file.go",
						NewLine: 7,
					},
				}},
			}}
			_ = json.NewEncoder(w).Encode(dls)
		}
	})
	mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14/discussions/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		resolved.Store(parts[len(parts)-1], true)
		_ = json.NewEncoder(w).Encode(gitlab.Discussion{})
	})
	mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"target_project_id": 14, "target_branch": "test-branch"}`))
	})
	mux.HandleFunc("/api/v4/projects/14/repository/branches/test-branch", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"commit": {"id": "xxx"}}`))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli, err := gitlab.NewClient("", gitlab.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatal(err)
	}

	g := NewGitLabMergeRequestDiscussionCommenter(cli, "o", "r", 14, "sha", "tool-name")
	if err := g.Flush(context.Background()); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if _, ok := resolved.Load("disc-page2-fixed"); !ok {
		t.Errorf("expected disc-page2-fixed (page 2) to be resolved")
	}
}

func TestBuildSuggestions(t *testing.T) {
	tests := []struct {
		in   *reviewdog.Comment
		want string
	}{
		{
			in: &reviewdog.Comment{
				ToolName: "tool-name",
				Result: &filter.FilteredDiagnostic{
					Diagnostic: &rdf.Diagnostic{
						Message: "no suggestion",
					},
				},
			},
			want: "",
		},
		{
			in: buildTestComment(
				"one suggestion",
				[]*rdf.Suggestion{
					buildTestsSuggestion("line1-fixed\nline2-fixed", 10, 10),
				},
			),
			want: strings.Join([]string{
				"```suggestion:-0+0",
				"line1-fixed",
				"line2-fixed",
				"```",
				"",
			}, "\n"),
		},
		{
			in: buildTestComment(
				"two suggestions",
				[]*rdf.Suggestion{
					buildTestsSuggestion("line1-fixed\nline2-fixed", 10, 11),
					buildTestsSuggestion("line3-fixed\nline4-fixed", 20, 21),
				},
			),
			want: strings.Join([]string{
				"```suggestion:-0+1",
				"line1-fixed",
				"line2-fixed",
				"```",
				"```suggestion:-0+1",
				"line3-fixed",
				"line4-fixed",
				"```",
				"",
			}, "\n"),
		},
		{
			in: buildTestComment(
				"a suggestion that has fenced code block",
				[]*rdf.Suggestion{
					buildTestsSuggestion("```shell\ngit config --global receive.advertisepushoptions true\n```", 10, 12),
				},
			),
			want: strings.Join([]string{
				"````suggestion:-0+2",
				"```shell",
				"git config --global receive.advertisepushoptions true",
				"```",
				"````",
				"",
			}, "\n"),
		},
	}
	for _, tt := range tests {
		suggestion := buildSuggestions(tt.in)
		if suggestion != tt.want {
			t.Errorf("got unexpected suggestion.\ngot:\n%s\nwant:\n%s", suggestion, tt.want)
		}
	}
}

func TestBuildSuggestionsInvalid(t *testing.T) {
	tests := []struct {
		in   *reviewdog.Comment
		want string
	}{
		{
			in: buildTestComment(
				"two suggestions, one without range",
				[]*rdf.Suggestion{
					{
						Text: "line3-fixed\nline4-fixed",
					},
					buildTestsSuggestion("line1-fixed\nline2-fixed", 10, 11),
				},
			),
			want: strings.Join([]string{
				"```suggestion:-0+1",
				"line1-fixed",
				"line2-fixed",
				"```",
				"",
			}, "\n"),
		},
		{
			in: buildTestComment(
				"two suggestions, one without range end",
				[]*rdf.Suggestion{
					{
						Text: "line3-fixed\nline4-fixed",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 20,
							},
						},
					},
					buildTestsSuggestion("line1-fixed\nline2-fixed", 10, 11),
				}),
			want: strings.Join([]string{
				"```suggestion:-0+1",
				"line1-fixed",
				"line2-fixed",
				"```",
				"",
			}, "\n"),
		},
	}
	for _, tt := range tests {
		suggestion := buildSuggestions(tt.in)
		if suggestion != tt.want {
			t.Errorf("got unexpected suggestion.\ngot:\n%s\nwant:\n%s", suggestion, tt.want)
		}
	}
}

func buildTestsSuggestion(text string, start int32, end int32) *rdf.Suggestion {
	return &rdf.Suggestion{
		Text: text,
		Range: &rdf.Range{
			Start: &rdf.Position{
				Line: start,
			},
			End: &rdf.Position{
				Line: end,
			},
		},
	}
}

func buildTestComment(message string, suggestions []*rdf.Suggestion) *reviewdog.Comment {
	return &reviewdog.Comment{
		ToolName: "tool-name",
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Message:     message,
				Suggestions: suggestions,
			},
		},
	}
}
