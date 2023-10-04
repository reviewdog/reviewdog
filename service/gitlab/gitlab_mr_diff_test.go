package gitlab

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xanzy/go-gitlab"
)

func TestGitLabMergeRequestDiff_Diff(t *testing.T) {
	getMRAPICall := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/o/r/merge_requests/14", func(w http.ResponseWriter, r *http.Request) {
		getMRAPICall++
		if r.Method != http.MethodGet {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		w.Write([]byte(`{"target_project_id": 14, "target_branch": "test-branch","diff_refs": {"base_sha": "HEAD~", "head_sha": "HEAD", "start_sha": "HEAD"}}`))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli, err := gitlab.NewClient("", gitlab.WithBaseURL(ts.URL+"/api/v4"))
	if err != nil {
		t.Fatal(err)
	}

	g, err := NewGitLabMergeRequestDiff(cli, "o", "r", 14, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Diff(context.Background()); err != nil {
		t.Fatal(err)
	}
	if getMRAPICall != 1 {
		t.Errorf("Get GitLab MergeRequest API called %v times, want once", getMRAPICall)
	}
}
