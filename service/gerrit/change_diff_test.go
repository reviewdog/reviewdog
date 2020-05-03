package gerrit

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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
