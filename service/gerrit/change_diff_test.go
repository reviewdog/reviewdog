package gerrit

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/build/gerrit"
)

func TestChangeDiff_Diff(t *testing.T) {
	getChangeDetailAPICall := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/changes/changeID/detail", func(w http.ResponseWriter, r *http.Request) {
		getChangeDetailAPICall++
		if r.Method != http.MethodGet {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}

		fmt.Fprintf(w, ")]}\n{\"current_revision\": \"HEAD\"}")
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := gerrit.NewClient(ts.URL, gerrit.NoAuth)

	g, err := NewChangeDiff(cli, "HEAD^", "changeID")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Diff(context.Background()); err != nil {
		t.Fatal(err)
	}
	if getChangeDetailAPICall != 1 {
		t.Errorf("Get GitLab MergeRequest API called %v times, want once", getChangeDetailAPICall)
	}
}

func TestChangeDiff_gitDiff_reportsGitError(t *testing.T) {
	g := &ChangeDiff{}
	_, err := g.gitDiff(context.Background(), "HEAD", "no-such-revision")
	if err == nil {
		t.Fatal("expected an error for an unknown revision")
	}
	if !strings.Contains(err.Error(), "no-such-revision") {
		t.Errorf("error should include git's stderr, got: %v", err)
	}
}
