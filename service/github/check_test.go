package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v60/github"
	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/doghouse"
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
		wantCheckID = 1414
	)

	comments := []*reviewdog.Comment{
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line:   15,
								Column: 14,
							},
						},
					},
					Message: "comment 1",
				},
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line:   1,
								Column: 2,
							},
							End: &rdf.Position{
								Line:   3,
								Column: 4,
							},
						},
					},
					Message: "comment 2",
					Suggestions: []*rdf.Suggestion{
						{
							Range: &rdf.Range{
								Start: &rdf.Position{Line: 15, Column: 5},
								End:   &rdf.Position{Line: 15, Column: 7},
							},
							Text: "14",
						},
					},
					RelatedLocations: []*rdf.RelatedLocation{
						{
							Location: &rdf.Location{
								Path: "reviewdog.go",
								Range: &rdf.Range{
									Start: &rdf.Position{
										Line: 14,
									},
									End: &rdf.Position{
										Line: 16,
									},
								},
							},
							Message: "related loc test",
						},
						{
							Location: &rdf.Location{
								Path: filepath.Join(cwd, "reviewdog2.go"),
								Range: &rdf.Range{
									Start: &rdf.Position{
										Line: 14,
									},
								},
							},
							Message: "related loc test (2)",
						},
					},
				},
			},
		},
	}

	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()

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
			},
			{
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
			},
			{
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
			},
			{
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
			},
			{
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
			},
			{
				Diagnostic: &rdf.Diagnostic{
					Message: "code test w/o URL",
					Location: &rdf.Location{
						Path:  "sample.new.txt",
						Range: &rdf.Range{Start: &rdf.Position{Line: 2}},
					},
					Code: &rdf.Code{Value: "CODE14"},
				},
			},
			{
				Diagnostic: &rdf.Diagnostic{
					Message: "code test w/ URL",
					Location: &rdf.Location{
						Path:  "sample.new.txt",
						Range: &rdf.Range{Start: &rdf.Position{Line: 2}},
					},
					Code: &rdf.Code{Value: "CODE14", Url: "https://github.com/reviewdog#CODE14"},
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

	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int, fallbackToGitCLI bool) ([]byte, error) {
		return []byte(sampleDiff), nil
	}
	cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
		if opt.Name != name {
			t.Errorf("CreateCheckRunOptions.Name = %q, want %q", opt.Name, name)
		}
		if opt.HeadSHA != sha {
			t.Errorf("CreateCheckRunOptions.HeadSHA = %q, want %q", opt.HeadSHA, sha)
		}
		return &github.CheckRun{ID: github.Int64(wantCheckID)}, nil
	}
	cli.FakeUpdateCheckRun = func(ctx context.Context, owner, repo string, checkID int64, opt github.UpdateCheckRunOptions) (*github.CheckRun, error) {
		if checkID != wantCheckID {
			t.Errorf("UpdateCheckRun: checkID = %d, want %d", checkID, wantCheckID)
		}
		if opt.Name != name {
			t.Errorf("UpdateCheckRunOptions.Name = %q, want %q", opt.Name, name)
		}
		annotations := opt.Output.Annotations
		if len(annotations) == 0 {
			if *opt.Conclusion != conclusion {
				t.Errorf("UpdateCheckRunOptions.Conclusion = %q, want %q", *opt.Conclusion, conclusion)
			}
		} else {
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
			if d := cmp.Diff(annotations, wantAnnotations); d != "" {
				t.Errorf("Annotation diff found:\n%s", d)
			}
		}
		return &github.CheckRun{HTMLURL: github.String(reportURL)}, nil
	}
	checker := &Checker{req: req, gh: cli}
	res, err := checker.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if res.ReportURL != reportURL {
		t.Errorf("res.reportURL = %q, want %q", res.ReportURL, reportURL)
	}
}
