package gitlab

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gitlab "gitlab.com/gitlab-org/api/client-go/v3"
)

func TestGitLabMergeRequestDiff_Diff(t *testing.T) {
	getMRAPICall := 0
	getBranchAPICall := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/o%2Fr/merge_requests/14", func(w http.ResponseWriter, r *http.Request) {
		getMRAPICall++
		if r.Method != http.MethodGet {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		w.Write([]byte(`{"target_project_id": 14, "target_branch": "test-branch"}`))
	})
	mux.HandleFunc("/api/v4/projects/14/repository/branches/test-branch", func(w http.ResponseWriter, r *http.Request) {
		getBranchAPICall++
		if r.Method != http.MethodGet {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		w.Write([]byte(`{"commit": {"id": "HEAD~"}}`))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli, err := gitlab.NewClient("", gitlab.WithBaseURL(ts.URL+"/api/v4"))
	if err != nil {
		t.Fatal(err)
	}

	g := NewGitLabMergeRequestDiff(cli, "o", "r", 14, "HEAD")
	if _, err := g.Diff(context.Background()); err != nil {
		t.Fatal(err)
	}
	if getMRAPICall != 1 {
		t.Errorf("Get GitLab MergeRequest API called %v times, want once", getMRAPICall)
	}
	if getBranchAPICall != 1 {
		t.Errorf("Get GitLab Branch API called %v times, want once", getBranchAPICall)
	}
}

func TestGitLabMergeRequestDiff_gitDiff_reportsGitError(t *testing.T) {
	tests := []struct {
		name          string
		diffBaseSha   string
		targetSha     string
		wantErrPrefix string
	}{
		{
			name:          "merge-base",
			targetSha:     "no-such-revision",
			wantErrPrefix: "failed to get merge-base commit",
		},
		{
			name:          "diff",
			diffBaseSha:   "no-such-revision",
			targetSha:     "HEAD~",
			wantErrPrefix: "failed to run git diff",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CI_MERGE_REQUEST_DIFF_BASE_SHA", tt.diffBaseSha)
			g := &MergeRequestDiff{}
			_, err := g.gitDiff(context.Background(), "HEAD", tt.targetSha)
			if err == nil {
				t.Fatal("expected an error for an unknown revision")
			}
			if !strings.HasPrefix(err.Error(), tt.wantErrPrefix) {
				t.Errorf("unexpected error: %v", err)
			}
			if !strings.Contains(err.Error(), "fatal:") {
				t.Errorf("error should include git's stderr, got: %v", err)
			}
		})
	}
}
