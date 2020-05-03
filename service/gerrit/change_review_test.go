package gerrit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/reviewdog/reviewdog"
	"golang.org/x/build/gerrit"
)

func TestChangeReviewCommenter_Post_Flush(t *testing.T) {
	cwd, _ := os.Getwd()
	defer func(dir string) {
		if err := os.Chdir(dir); err != nil {
			t.Error(err)
		}
	}(cwd)
	if err := os.Chdir("../.."); err != nil {
		t.Error(err)
	}

	ctx := context.Background()
	newLnum1 := 14
	newComment1 := &reviewdog.Comment{
		CheckResult: &reviewdog.CheckResult{
			Path: "file.go",
			Lnum: newLnum1,
		},
		Body: "new comment",
	}
	newLnum2 := 15
	newComment2 := &reviewdog.Comment{
		CheckResult: &reviewdog.CheckResult{
			Path: "file2.go",
			Lnum: newLnum2,
		},
		Body: "new comment 2",
	}

	comments := []*reviewdog.Comment{
		newComment1,
		newComment2,
	}

	mux := http.NewServeMux()
	mux.HandleFunc(`/changes/testChangeID/revisions/testRevisionID/review`, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			got := new(gerrit.ReviewInput)
			if err := json.NewDecoder(r.Body).Decode(got); err != nil {
				t.Error(err)
			}

			if len(got.Comments) != len(comments) {
				t.Error("expected two comments")
			}

			want := []gerrit.CommentInput{{Line: newComment1.Lnum, Message: newComment1.Body}}
			if diff := cmp.Diff(got.Comments["file.go"], want); diff != "" {
				t.Error(diff)
			}

			want = []gerrit.CommentInput{{Line: newComment2.Lnum, Message: newComment2.Body}}
			if diff := cmp.Diff(got.Comments["file2.go"], want); diff != "" {
				t.Error(diff)
			}

			fmt.Fprintf(w, ")]}\n{}")
		default:
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := gerrit.NewClient(ts.URL, gerrit.NoAuth)

	g, err := NewChangeReviewCommenter(cli, "testChangeID", "testRevisionID")
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range comments {
		if err := g.Post(ctx, c); err != nil {
			t.Error(err)
		}
	}

	if err := g.Flush(ctx); err != nil {
		t.Errorf("%v", err)
	}
}
