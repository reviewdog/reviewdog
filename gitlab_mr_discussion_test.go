package reviewdog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	gitlab "github.com/xanzy/go-gitlab"
)

func TestGitLabMergeRequestDiscussionCommenter_Post_Flush_review_api(t *testing.T) {
	alreadyCommented1 := &Comment{
		CheckResult: &CheckResult{
			Path: "file.go",
			Lnum: 1,
		},
		Body: "already commented",
	}
	alreadyCommented2 := &Comment{
		CheckResult: &CheckResult{
			Path: "another/file.go",
			Lnum: 14,
		},
		Body: "already commented 2",
	}
	newComment1 := &Comment{
		CheckResult: &CheckResult{
			Path: "file.go",
			Lnum: 14,
		},
		Body: "new comment",
	}
	newComment2 := &Comment{
		CheckResult: &CheckResult{
			Path: "file2.go",
			Lnum: 15,
		},
		Body: "new comment 2",
	}

	comments := []*Comment{
		alreadyCommented1,
		alreadyCommented2,
		newComment1,
		newComment2,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/o/r/merge_requests/14/discussions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			switch r.URL.Query().Get("page") {
			default:
				dls := []*GitLabMergeRequestDiscussionList{
					{
						Notes: []*GitLabMergeRequestDiscussion{
							{
								Body: commentBody(alreadyCommented1),
								Position: &GitLabMergeRequestDiscussionPosition{
									NewPath: alreadyCommented1.Path,
									NewLine: alreadyCommented1.Lnum,
								},
							},
							{
								Body: "unrelated commented",
								Position: &GitLabMergeRequestDiscussionPosition{
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
				dls := []*GitLabMergeRequestDiscussionList{
					{
						Notes: []*GitLabMergeRequestDiscussion{
							{
								Body: commentBody(alreadyCommented2),
								Position: &GitLabMergeRequestDiscussionPosition{
									NewPath: alreadyCommented2.Path,
									NewLine: alreadyCommented2.Lnum,
								},
							},
						},
					},
				}
				if err := json.NewEncoder(w).Encode(dls); err != nil {
					t.Fatal(err)
				}
			}

		case "POST":
			got := new(GitLabMergeRequestDiscussion)
			if err := json.NewDecoder(r.Body).Decode(got); err != nil {
				t.Error(err)
			}
			switch got.Position.NewPath {
			case "file.go":
				want := &GitLabMergeRequestDiscussion{
					Body: commentBody(newComment1),
					Position: &GitLabMergeRequestDiscussionPosition{
						BaseSHA: "sha", StartSHA: "xxx", HeadSHA: "sha", PositionType: "text", NewPath: "file.go", NewLine: 14},
				}
				if diff := cmp.Diff(got, want); diff != "" {
					t.Error(diff)
				}
			case "file2.go":
				want := &GitLabMergeRequestDiscussion{
					Body: commentBody(newComment2),
					Position: &GitLabMergeRequestDiscussionPosition{
						BaseSHA: "sha", StartSHA: "xxx", HeadSHA: "sha", PositionType: "text", NewPath: "file2.go", NewLine: 15},
				}
				if diff := cmp.Diff(got, want); diff != "" {
					t.Error(diff)
				}
			default:
				t.Errorf("got unexpected discussion: %#v", got)
			}
		default:
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
	})
	mux.HandleFunc("/api/v4/projects/o/r/merge_requests/14", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		w.Write([]byte(`{"target_project_id": 14, "target_branch": "test-branch"}`))
	})
	mux.HandleFunc("/api/v4/projects/14/repository/branches/test-branch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		w.Write([]byte(`{"commit": {"id": "xxx"}}`))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := gitlab.NewClient(nil, "")
	cli.SetBaseURL(ts.URL + "/api/v4")

	g, err := NewGitLabMergeRequestDiscussionCommenter(cli, "o", "r", 14, "sha")
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range comments {
		if err := g.Post(context.Background(), c); err != nil {
			t.Error(err)
		}
	}
	if err := g.Flush(context.Background()); err != nil {
		t.Error(err)
	}
}
