package reviewdog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gitlab "github.com/xanzy/go-gitlab"
)

func TestGitLabMergeRequestDiff_Diff(t *testing.T) {
	getMRAPICall := 0
	getBranchAPICall := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/o/r/merge_requests/14", func(w http.ResponseWriter, r *http.Request) {
		getMRAPICall++
		if r.Method != "GET" {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		w.Write([]byte(`{"target_project_id": 14, "target_branch": "test-branch"}`))
	})
	mux.HandleFunc("/api/v4/projects/14/repository/branches/test-branch", func(w http.ResponseWriter, r *http.Request) {
		getBranchAPICall++
		if r.Method != "GET" {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		w.Write([]byte(`{"commit": {"id": "master"}}`))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := gitlab.NewClient(nil, "")
	cli.SetBaseURL(ts.URL + "/api/v4")

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
	if getBranchAPICall != 1 {
		t.Errorf("Get GitLab Branch API called %v times, want once", getBranchAPICall)
	}
}
