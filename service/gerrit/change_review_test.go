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
	"golang.org/x/build/gerrit"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
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
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: "file.go",
					Range: &rdf.Range{Start: &rdf.Position{
						Line: int32(newLnum1),
					}},
				},
				Message: "new comment",
			},
			InDiffFile: true,
		},
	}
	newLnum2 := 15
	newComment2 := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: "file2.go",
					Range: &rdf.Range{Start: &rdf.Position{
						Line: int32(newLnum2),
					}},
				},
				Message: "new comment 2",
			},
			InDiffFile: true,
		},
	}
	commentOutsideDiff := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: "file3.go",
					Range: &rdf.Range{Start: &rdf.Position{
						Line: 14,
					}},
				},
				Message: "comment outside diff",
			},
			InDiffFile: false,
		},
	}

	comments := []*reviewdog.Comment{
		newComment1,
		newComment2,
		commentOutsideDiff,
	}

	mux := http.NewServeMux()
	mux.HandleFunc(`/changes/testChangeID/revisions/testRevisionID/review`, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			got := new(gerrit.ReviewInput)
			if err := json.NewDecoder(r.Body).Decode(got); err != nil {
				t.Error(err)
			}

			if want := len(comments) - 1; len(got.Comments) != want {
				t.Errorf("got %d comments, want %d", len(got.Comments), want)
			}

			line1 := int(newComment1.Result.Diagnostic.GetLocation().GetRange().GetStart().GetLine())
			want := []gerrit.CommentInput{{
				Line: line1, Message: newComment1.Result.Diagnostic.GetMessage()}}
			if diff := cmp.Diff(got.Comments["file.go"], want); diff != "" {
				t.Error(diff)
			}

			line2 := int(newComment2.Result.Diagnostic.GetLocation().GetRange().GetStart().GetLine())
			want = []gerrit.CommentInput{{
				Line: line2, Message: newComment2.Result.Diagnostic.GetMessage()}}
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
