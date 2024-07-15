package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v63/github"
	"github.com/reviewdog/reviewdog/doghouse"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

const sampleDiff = `--- a/sample.old.txt	2016-10-13 05:09:35.820791185 +0900
+++ b/sample.new.txt	2016-10-13 05:15:26.839245048 +0900
@@ -1,3 +1,4 @@
 unchanged, contextual line
-deleted line
+added line
+added line
 unchanged, contextual line
--- a/nonewline.old.txt	2016-10-13 15:34:14.931778318 +0900
+++ b/nonewline.new.txt	2016-10-13 15:34:14.868444672 +0900
@@ -1,4 +1,4 @@
 " vim: nofixeol noendofline
 No newline at end of both the old and new file
-a
-a
\ No newline at end of file
+b
+b
\ No newline at end of file
`

func TestCheck_OK(t *testing.T) {
	const (
		name        = "haya14busa-linter"
		owner       = "haya14busa"
		repo        = "reviewdog"
		prNum       = 14
		sha         = "1414"
		reportURL   = "http://example.com/report_url"
		conclusion  = "neutral"
		wantCheckID = 1414
	)

	req := &doghouse.CheckRequest{
		Name:        name,
		Owner:       owner,
		Repo:        repo,
		PullRequest: prNum,
		SHA:         sha,
		Annotations: []*doghouse.Annotation{
			{
				Diagnostic: &rdf.Diagnostic{
					Message: "test message",
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 2, Column: 1},
						},
					},
					OriginalOutput: "raw test message",
				},
			},
			{
				Diagnostic: &rdf.Diagnostic{
					Message: "test message outside diff",
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 14},
						},
					},
					OriginalOutput: "raw test message outside diff",
				},
			},
			{
				Path:       "sample.new.txt",
				Line:       2,
				Message:    "request from old clients",
				RawMessage: "raw message from old clients",
			},
		},
		Level: "warning",
	}

	// cli := &fakeCheckerGitHubCli{}
	// cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int, fallbackToGitCLI bool) ([]byte, error) {
	// 	return []byte(sampleDiff), nil
	// }
	// cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
	// 	if opt.Name != name {
	// 		t.Errorf("CreateCheckRunOptions.Name = %q, want %q", opt.Name, name)
	// 	}
	// 	if opt.HeadSHA != sha {
	// 		t.Errorf("CreateCheckRunOptions.HeadSHA = %q, want %q", opt.HeadSHA, sha)
	// 	}
	// 	return &github.CheckRun{ID: github.Int64(wantCheckID)}, nil
	// }
	// cli.FakeUpdateCheckRun = func(ctx context.Context, owner, repo string, checkID int64, opt github.UpdateCheckRunOptions) (*github.CheckRun, error) {
	// 	if checkID != wantCheckID {
	// 		t.Errorf("UpdateCheckRun: checkID = %d, want %d", checkID, wantCheckID)
	// 	}
	// 	if opt.Name != name {
	// 		t.Errorf("UpdateCheckRunOptions.Name = %q, want %q", opt.Name, name)
	// 	}
	// 	annotations := opt.Output.Annotations
	// 	if len(annotations) == 0 {
	// 		if *opt.Conclusion != conclusion {
	// 			t.Errorf("UpdateCheckRunOptions.Conclusion = %q, want %q", *opt.Conclusion, conclusion)
	// 		}
	// 	} else {
	// 		wantAnnotations := []*github.CheckRunAnnotation{
	// 			{
	// 				Path:            github.String("sample.new.txt"),
	// 				StartLine:       github.Int(2),
	// 				EndLine:         github.Int(2),
	// 				AnnotationLevel: github.String("warning"),
	// 				Message:         github.String("test message"),
	// 				Title:           github.String("[haya14busa-linter] sample.new.txt#L2"),
	// 				RawDetails:      github.String("raw test message"),
	// 			},
	// 			{
	// 				Path:            github.String("sample.new.txt"),
	// 				StartLine:       github.Int(2),
	// 				EndLine:         github.Int(3),
	// 				AnnotationLevel: github.String("warning"),
	// 				Message:         github.String("test multiline"),
	// 				Title:           github.String("[haya14busa-linter] sample.new.txt#L2-L3"),
	// 			},
	// 			{
	// 				Path:            github.String("sample.new.txt"),
	// 				StartLine:       github.Int(2),
	// 				EndLine:         github.Int(3),
	// 				AnnotationLevel: github.String("warning"),
	// 				Message:         github.String("test multiline with column"),
	// 				Title:           github.String("[haya14busa-linter] sample.new.txt#L2-L3"),
	// 			},
	// 			{
	// 				Path:            github.String("sample.new.txt"),
	// 				StartLine:       github.Int(2),
	// 				EndLine:         github.Int(2),
	// 				StartColumn:     github.Int(1),
	// 				EndColumn:       github.Int(5),
	// 				AnnotationLevel: github.String("warning"),
	// 				Message:         github.String("test range comment"),
	// 				Title:           github.String("[haya14busa-linter] sample.new.txt#L2"),
	// 			},
	// 			{
	// 				Path:            github.String("sample.new.txt"),
	// 				StartLine:       github.Int(2),
	// 				EndLine:         github.Int(2),
	// 				AnnotationLevel: github.String("failure"),
	// 				Message:         github.String("test severity override"),
	// 				Title:           github.String("[haya14busa-linter] sample.new.txt#L2"),
	// 			},
	// 			{
	// 				Path:            github.String("sample.new.txt"),
	// 				StartLine:       github.Int(2),
	// 				EndLine:         github.Int(2),
	// 				AnnotationLevel: github.String("warning"),
	// 				Message:         github.String("source test"),
	// 				Title:           github.String("[awesome-linter] sample.new.txt#L2"),
	// 			},
	// 			{
	// 				Path:            github.String("sample.new.txt"),
	// 				StartLine:       github.Int(2),
	// 				EndLine:         github.Int(2),
	// 				AnnotationLevel: github.String("warning"),
	// 				Message:         github.String("code test w/o URL"),
	// 				Title:           github.String("[haya14busa-linter] sample.new.txt#L2 <CODE14>"),
	// 			},
	// 			{
	// 				Path:            github.String("sample.new.txt"),
	// 				StartLine:       github.Int(2),
	// 				EndLine:         github.Int(2),
	// 				AnnotationLevel: github.String("warning"),
	// 				Message:         github.String("code test w/ URL"),
	// 				Title:           github.String("[haya14busa-linter] sample.new.txt#L2 <CODE14>(https://github.com/reviewdog#CODE14)"),
	// 			},
	// 			{
	// 				Path:            github.String("sample.new.txt"),
	// 				StartLine:       github.Int(2),
	// 				EndLine:         github.Int(2),
	// 				AnnotationLevel: github.String("warning"),
	// 				Message:         github.String("request from old clients"),
	// 				Title:           github.String("[haya14busa-linter] sample.new.txt#L2"),
	// 				RawDetails:      github.String("raw message from old clients"),
	// 			},
	// 		}
	// 		if d := cmp.Diff(annotations, wantAnnotations); d != "" {
	// 			t.Errorf("Annotation diff found:\n%s", d)
	// 		}
	// 	}
	// 	return &github.CheckRun{HTMLURL: github.String(reportURL)}, nil
	// }
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/haya14busa/reviewdog/pulls/14", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleDiff))
	})
	mux.HandleFunc("/repos/haya14busa/reviewdog/check-runs", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(&github.CheckRun{ID: github.Int64(wantCheckID)}); err != nil {
			t.Fatal(err)
		}
	})
	mux.HandleFunc("/repos/haya14busa/reviewdog/check-runs/1414", func(w http.ResponseWriter, r *http.Request) {
		var req github.CheckRun
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		wantAnnotations := []*github.CheckRunAnnotation{
			{
				Path:            github.String("sample.new.txt"),
				StartLine:       github.Int(2),
				EndLine:         github.Int(2),
				AnnotationLevel: github.String("warning"),
				Message:         github.String("test message"),
				Title:           github.String("[haya14busa-linter] sample.new.txt#L2"),
				RawDetails:      github.String("raw test message"),
			},
			{
				Path:            github.String("sample.new.txt"),
				StartLine:       github.Int(2),
				EndLine:         github.Int(2),
				AnnotationLevel: github.String("warning"),
				Message:         github.String("request from old clients"),
				Title:           github.String("[haya14busa-linter] sample.new.txt#L2"),
				RawDetails:      github.String("raw message from old clients"),
			},
		}
		if req.GetStatus() != "completed" {
			if d := cmp.Diff(req.Output.Annotations, wantAnnotations); d != "" {
				t.Errorf("Annotation diff found:\n%s", d)
			}
		}
		if err := json.NewEncoder(w).Encode(&github.CheckRun{HTMLURL: github.String(reportURL)}); err != nil {
			t.Fatal(err)
		}
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")
	checker := NewChecker(req, cli, true /* inDogHouseServer */)
	res, err := checker.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.ReportURL != reportURL {
		t.Errorf("res.reportURL = %q, want %q", res.ReportURL, reportURL)
	}
}
