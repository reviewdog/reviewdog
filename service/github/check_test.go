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
	"github.com/google/go-github/v63/github"
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

		if req.GetStatus() == "completed" {
			if wantTitle := "reviewdog [haya14busa-linter] report"; req.GetOutput().GetTitle() != wantTitle {
				t.Errorf("title = %s, want %s", req.GetOutput().GetTitle(), wantTitle)
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

func TestCheck_setToolNameForEachRun(t *testing.T) {
	const (
		owner       = "haya14busa"
		repo        = "reviewdog"
		prNum       = 14
		sha         = "1414"
		reportURL   = "http://example.com/report_url"
		conclusion  = "neutral"
		wantCheckID = 1414
		toolName1   = "toolName1"
		level1      = "warning"
		toolName2   = "toolName2"
		level2      = ""
	)

	mux := http.NewServeMux()
	checkRunNum := 0
	mux.HandleFunc("/repos/haya14busa/reviewdog/check-runs", func(w http.ResponseWriter, r *http.Request) {
		checkRunNum++
		var req github.CreateCheckRunOptions
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		switch checkRunNum {
		case 1:
			if req.Name != toolName1 {
				t.Errorf("toolName = %s, want %s", req.Name, toolName1)
			}
		case 2:
			if req.Name != toolName2 {
				t.Errorf("toolName = %s, want %s", req.Name, toolName2)
			}
		}
		if err := json.NewEncoder(w).Encode(&github.CheckRun{ID: github.Int64(wantCheckID)}); err != nil {
			t.Fatal(err)
		}
	})
	updateCheckRunNum := 0
	mux.HandleFunc("/repos/haya14busa/reviewdog/check-runs/1414", func(w http.ResponseWriter, r *http.Request) {
		var req github.CheckRun
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		switch updateCheckRunNum {
		case 0:
			if req.GetName() != toolName1 {
				t.Errorf("toolName = %s, want %s", req.GetName(), toolName1)
			}
			wantAnnotations := []*github.CheckRunAnnotation{
				{
					Path:            github.String("sample.new.txt"),
					StartLine:       github.Int(2),
					EndLine:         github.Int(2),
					AnnotationLevel: github.String("warning"),
					Message:         github.String("comment 1"),
					Title:           github.String("[toolName1] sample.new.txt#L2"),
				},
			}
			if req.GetStatus() == "completed" {
				if wantTitle := "reviewdog [toolName1] report"; req.GetOutput().GetTitle() != wantTitle {
					t.Errorf("title = %s, want %s", req.GetOutput().GetTitle(), wantTitle)
				}
			} else {
				if d := cmp.Diff(req.Output.Annotations, wantAnnotations); d != "" {
					t.Errorf("Annotation diff found:\n%s", d)
				}
			}
		case 1:
			if req.GetName() != toolName2 {
				t.Errorf("toolName = %s, want %s", req.GetName(), toolName2)
			}
			wantAnnotations := []*github.CheckRunAnnotation{
				{
					Path:            github.String("sample.new.txt"),
					StartLine:       github.Int(2),
					EndLine:         github.Int(2),
					AnnotationLevel: github.String("failure"), // default
					Message:         github.String("comment 2"),
					Title:           github.String("[toolName2] sample.new.txt#L2"),
				},
			}
			if req.GetStatus() == "completed" {
				if wantTitle := "reviewdog [toolName2] report"; req.GetOutput().GetTitle() != wantTitle {
					t.Errorf("title = %s, want %s", req.GetOutput().GetTitle(), wantTitle)
				}
			} else {
				if d := cmp.Diff(req.Output.Annotations, wantAnnotations); d != "" {
					t.Errorf("Annotation diff found:\n%s", d)
				}
			}
		}
		if req.GetStatus() == "completed" {
			updateCheckRunNum++
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
		ToolName: "", // empty
		Level:    "", // empty
	}

	check.SetTool(toolName1, level1)
	if err := check.Post(context.Background(), &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Message: "comment 1",
				Location: &rdf.Location{
					Path: "sample.new.txt",
					Range: &rdf.Range{
						Start: &rdf.Position{Line: 2, Column: 1},
					},
				},
			},
			ShouldReport: true,
		},
	}); err != nil {
		t.Errorf("failed to post: %v", err)
	}
	if err := check.Flush(context.Background()); err != nil {
		t.Error(err)
	}

	check.SetTool(toolName2, level2)
	if err := check.Post(context.Background(), &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Message: "comment 2",
				Location: &rdf.Location{
					Path: "sample.new.txt",
					Range: &rdf.Range{
						Start: &rdf.Position{Line: 2, Column: 1},
					},
				},
			},
			ShouldReport: true,
		},
	}); err != nil {
		t.Errorf("failed to post: %v", err)
	}
	if err := check.Flush(context.Background()); err != nil {
		t.Error(err)
	}
}
