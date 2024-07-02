package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v62/github"
	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

func TestCheck_OK(t *testing.T) {
	const (
		name        = "haya14busa-linter"
		owner       = "haya14busa"
		repo        = "reviewdog"
		prNum       = 14
		sha         = "1414"
		reportURL   = "http://example.com/report_url"
		conclusion  = "neutral"
		level       = "warning"
		wantCheckID = 1414
	)

	comments := []*reviewdog.Comment{
		{
			Result: &filter.FilteredDiagnostic{
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
				ShouldReport: true,
			},
		}, {
			Result: &filter.FilteredDiagnostic{
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
				ShouldReport: false,
			},
		}, {
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Message: "test multiline",
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 2},
							End:   &rdf.Position{Line: 3},
						},
					},
				},
				ShouldReport: true,
			},
		}, {
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Message: "test multiline with column",
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 2, Column: 1},
							End:   &rdf.Position{Line: 3, Column: 5},
						},
					},
				},
				ShouldReport: true,
			},
		}, {
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Message: "test range comment",
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 2, Column: 1},
							End:   &rdf.Position{Line: 2, Column: 5},
						},
					},
				},
				ShouldReport: true,
			},
		}, {
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Message:  "test severity override",
					Severity: rdf.Severity_ERROR,
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 2},
						},
					},
				},
				ShouldReport: true,
			},
		}, {
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Message: "source test",
					Source: &rdf.Source{
						Name: "awesome-linter",
					},
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 2},
						},
					},
				},
				ShouldReport: true,
			},
		}, {
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Message: "code test w/o URL",
					Location: &rdf.Location{
						Path:  "sample.new.txt",
						Range: &rdf.Range{Start: &rdf.Position{Line: 2}},
					},
					Code: &rdf.Code{Value: "CODE14"},
				},
				ShouldReport: true,
			},
		}, {
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Message: "code test w/ URL",
					Location: &rdf.Location{
						Path:  "sample.new.txt",
						Range: &rdf.Range{Start: &rdf.Position{Line: 2}},
					},
					Code: &rdf.Code{Value: "CODE14", Url: "https://github.com/reviewdog#CODE14"},
				},
				ShouldReport: true,
			},
		},
	}

	mux := http.NewServeMux()
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
				EndLine:         github.Int(3),
				AnnotationLevel: github.String("warning"),
				Message:         github.String("test multiline"),
				Title:           github.String("[haya14busa-linter] sample.new.txt#L2-L3"),
			},
			{
				Path:            github.String("sample.new.txt"),
				StartLine:       github.Int(2),
				EndLine:         github.Int(3),
				AnnotationLevel: github.String("warning"),
				Message:         github.String("test multiline with column"),
				Title:           github.String("[haya14busa-linter] sample.new.txt#L2-L3"),
			},
			{
				Path:            github.String("sample.new.txt"),
				StartLine:       github.Int(2),
				EndLine:         github.Int(2),
				StartColumn:     github.Int(1),
				EndColumn:       github.Int(5),
				AnnotationLevel: github.String("warning"),
				Message:         github.String("test range comment"),
				Title:           github.String("[haya14busa-linter] sample.new.txt#L2"),
			},
			{
				Path:            github.String("sample.new.txt"),
				StartLine:       github.Int(2),
				EndLine:         github.Int(2),
				AnnotationLevel: github.String("failure"),
				Message:         github.String("test severity override"),
				Title:           github.String("[haya14busa-linter] sample.new.txt#L2"),
			},
			{
				Path:            github.String("sample.new.txt"),
				StartLine:       github.Int(2),
				EndLine:         github.Int(2),
				AnnotationLevel: github.String("warning"),
				Message:         github.String("source test"),
				Title:           github.String("[awesome-linter] sample.new.txt#L2"),
			},
			{
				Path:            github.String("sample.new.txt"),
				StartLine:       github.Int(2),
				EndLine:         github.Int(2),
				AnnotationLevel: github.String("warning"),
				Message:         github.String("code test w/o URL"),
				Title:           github.String("[haya14busa-linter] sample.new.txt#L2 <CODE14>"),
			},
			{
				Path:            github.String("sample.new.txt"),
				StartLine:       github.Int(2),
				EndLine:         github.Int(2),
				AnnotationLevel: github.String("warning"),
				Message:         github.String("code test w/ URL"),
				Title:           github.String("[haya14busa-linter] sample.new.txt#L2 <CODE14>(https://github.com/reviewdog#CODE14)"),
			},
		}

		if req.GetStatus() == "completed" {
			if want := "neutral"; req.GetConclusion() != want {
				t.Errorf("conclusion = %s, want %s", req.GetConclusion(), want)
			}
			if wantTitle := "reviewdog [haya14busa-linter] report"; req.GetOutput().GetTitle() != wantTitle {
				t.Errorf("title = %s, want %s", req.GetOutput().GetTitle(), wantTitle)
			}
			if wantSummary := `reported by [reviewdog](https://github.com/reviewdog/reviewdog) :dog:
<details>
<summary>Findings (8)</summary>

[sample.new.txt|2 col 1|](https://github.com/haya14busa/reviewdog/blob/1414/sample.new.txt#L2) test message
[sample.new.txt|2|](https://github.com/haya14busa/reviewdog/blob/1414/sample.new.txt#L2) test multiline
[sample.new.txt|2 col 1|](https://github.com/haya14busa/reviewdog/blob/1414/sample.new.txt#L2) test multiline with column
[sample.new.txt|2 col 1|](https://github.com/haya14busa/reviewdog/blob/1414/sample.new.txt#L2) test range comment
[sample.new.txt|2|](https://github.com/haya14busa/reviewdog/blob/1414/sample.new.txt#L2) test severity override
[sample.new.txt|2|](https://github.com/haya14busa/reviewdog/blob/1414/sample.new.txt#L2) source test
[sample.new.txt|2|](https://github.com/haya14busa/reviewdog/blob/1414/sample.new.txt#L2) code test w/o URL
[sample.new.txt|2|](https://github.com/haya14busa/reviewdog/blob/1414/sample.new.txt#L2) code test w/ URL
</details>
<details>
<summary>Filtered Findings (1)</summary>

[sample.new.txt|14|](https://github.com/haya14busa/reviewdog/blob/1414/sample.new.txt#L14) test message outside diff
</details>`; req.GetOutput().GetSummary() != wantSummary {
				t.Errorf("summary =\n%s\n\nwant\n%s", req.GetOutput().GetSummary(), wantSummary)
			}
		} else {
			if d := cmp.Diff(req.Output.Annotations, wantAnnotations); d != "" {
				t.Errorf("Annotation diff found:\n%s", d)
			}
		}
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")

	check := &Check{
		CLI:      cli,
		Owner:    owner,
		Repo:     repo,
		PR:       prNum,
		SHA:      sha,
		ToolName: name,
		Level:    level,
	}

	for _, c := range comments {
		if c.Result.ShouldReport {
			if err := check.Post(context.Background(), c); err != nil {
				t.Errorf("failed to post: error=%v\ncomment=%v", err, c)
			}
		} else {
			if err := check.PostFiltered(context.Background(), c); err != nil {
				t.Errorf("failed to post: error=%v\ncomment=%v", err, c)
			}
		}
	}
	if err := check.Flush(context.Background()); err != nil {
		t.Error(err)
	}
	if check.GetResult().Conclusion != conclusion {
		t.Errorf("conclusion = %s, want %s", check.GetResult().Conclusion, conclusion)
	}
}

func TestCheck_OK_multiple_update_runs(t *testing.T) {
	const (
		name        = "haya14busa-linter"
		owner       = "haya14busa"
		repo        = "reviewdog"
		prNum       = 14
		sha         = "1414"
		reportURL   = "http://example.com/report_url"
		conclusion  = "neutral"
		level       = "warning"
		wantCheckID = 1414
	)

	mux := http.NewServeMux()
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
		switch len(req.GetOutput().Annotations) {
		case 0:
			if req.GetConclusion() != conclusion {
				t.Errorf("UpdateCheckRunOptions.Conclusion = %q, want %q", req.GetConclusion(), conclusion)
			}
		case maxAnnotationsPerRequest, 1: // Expected
		default:
			t.Errorf("UpdateCheckRun: len(annotations) = %d, but it's unexpected", len(req.GetOutput().Annotations))
		}
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")

	check := &Check{
		CLI:      cli,
		Owner:    owner,
		Repo:     repo,
		PR:       prNum,
		SHA:      sha,
		ToolName: name,
		Level:    level,
	}

	for i := 0; i < 101; i++ {
		if err := check.Post(context.Background(), &reviewdog.Comment{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Message: "test message",
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 2},
						},
					},
					OriginalOutput: "raw test message",
				},
				ShouldReport: true,
			},
		}); err != nil {
			t.Error(err)
		}
	}
	if err := check.Flush(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestCheck_fail_check_with_403_in_GitHub_Actions(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	const (
		name        = "haya14busa-linter"
		owner       = "haya14busa"
		repo        = "reviewdog"
		prNum       = 14
		sha         = "1414"
		reportURL   = "http://example.com/report_url"
		conclusion  = "neutral"
		level       = "warning"
		wantCheckID = 1414
	)
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/haya14busa/reviewdog/check-runs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")
	check := &Check{
		CLI:      cli,
		Owner:    owner,
		Repo:     repo,
		PR:       prNum,
		SHA:      sha,
		ToolName: name,
		Level:    level,
	}
	if err := check.Flush(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestCheck_summary_too_many_findings_cut_off_correctly(t *testing.T) {
	check := &Check{}
	var comment []*reviewdog.Comment
	for i := 0; i < 1000; i++ {
		comment = append(comment, &reviewdog.Comment{
			Result: &filter.FilteredDiagnostic{
				ShouldReport: true,
				Diagnostic: &rdf.Diagnostic{
					Message: "this is a pretty long test message that will lead to overshooting the maximum allowed size",
				},
			}})
	}
	summaryText := check.summary(comment)
	if len(summaryText) > maxAllowedSize {
		t.Errorf("summary text is %d bytes long, but the maximum allowed size is %d", len(summaryText), maxAllowedSize)
	}
	if !strings.Contains(summaryText, "... (Too many findings. Dropped some findings)\n</details>") {
		t.Error("summary text was not cut off correctly")
	}
}
