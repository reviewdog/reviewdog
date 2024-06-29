package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/cienv"
	"github.com/reviewdog/reviewdog/doghouse"
	"github.com/reviewdog/reviewdog/doghouse/client"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/project"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

func TestDiagnosticResultSet_Project(t *testing.T) {
	defer func(f func(ctx context.Context, conf *project.Config, runners map[string]bool, level string, tee bool) (*reviewdog.ResultMap, error)) {
		projectRunAndParse = f
	}(projectRunAndParse)

	var wantDiagnosticResult reviewdog.ResultMap
	wantDiagnosticResult.Store("name1", &reviewdog.Result{Diagnostics: []*rdf.Diagnostic{
		{
			Location: &rdf.Location{
				Range: &rdf.Range{Start: &rdf.Position{
					Line:   1,
					Column: 14,
				}},
				Path: "reviewdog.go",
			},
			Message: "msg",
		},
	}})

	projectRunAndParse = func(ctx context.Context, conf *project.Config, runners map[string]bool, level string, tee bool) (*reviewdog.ResultMap, error) {
		return &wantDiagnosticResult, nil
	}

	tmp, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	got, err := checkResultSet(context.Background(), nil, &option{conf: tmp.Name()}, true)
	if err != nil {
		t.Fatal(err)
	}

	if got.Len() != wantDiagnosticResult.Len() {
		t.Errorf("length of results is different. got = %d, want = %d\n", got.Len(), wantDiagnosticResult.Len())
	}
	got.Range(func(k string, r *reviewdog.Result) {
		w, _ := wantDiagnosticResult.Load(k)
		if diff := cmp.Diff(r, w, protocmp.Transform()); diff != "" {
			t.Errorf("result has diff:\n%s", diff)
		}
	})
}

func TestDiagnosticResultSet_NonProject(t *testing.T) {
	opt := &option{
		f: "golint",
	}
	input := `reviewdog.go:14:14: test message`
	got, err := checkResultSet(context.Background(), strings.NewReader(input), opt, false)
	if err != nil {
		t.Fatal(err)
	}
	var want reviewdog.ResultMap
	want.Store("golint", &reviewdog.Result{Diagnostics: []*rdf.Diagnostic{
		{
			Location: &rdf.Location{
				Range: &rdf.Range{Start: &rdf.Position{
					Line:   14,
					Column: 14,
				}},
				Path: "reviewdog.go",
			},
			Message:        "test message",
			OriginalOutput: input,
		},
	}})

	if got.Len() != want.Len() {
		t.Errorf("length of results is different. got = %d, want = %d\n", got.Len(), want.Len())
	}
	got.Range(func(k string, r *reviewdog.Result) {
		w, _ := want.Load(k)
		if diff := cmp.Diff(r, w, protocmp.Transform()); diff != "" {
			t.Errorf("result has diff:\n%s", diff)
		}
	})
}

func TestPostResultSet_withReportURL(t *testing.T) {
	const (
		owner = "haya14busa"
		repo  = "reviewdog"
		prNum = 14
		sha   = "1414"
	)
	mux := http.NewServeMux()
	mux.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method=%s", r.Method)
		}
		var req doghouse.CheckRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if req.Owner != owner {
			t.Errorf("req.Owner = %q, want %q", req.Owner, owner)
		}
		if req.Repo != repo {
			t.Errorf("req.Repo = %q, want %q", req.Repo, repo)
		}
		if req.SHA != sha {
			t.Errorf("req.SHA = %q, want %q", req.SHA, sha)
		}
		if req.PullRequest != prNum {
			t.Errorf("req.PullRequest = %d, want %d", req.PullRequest, prNum)
		}
		switch req.Name {
		case "name1":
			if diff := cmp.Diff(req.Annotations, []*doghouse.Annotation{
				{
					Diagnostic: &rdf.Diagnostic{
						Message: "name1: test 1",
						Location: &rdf.Location{
							Path: "cmd/reviewdog/reviewdog.go",
							Range: &rdf.Range{
								Start: &rdf.Position{Line: 14},
							},
						},
						OriginalOutput: "L1\nL2",
					},
				},
				{
					Diagnostic: &rdf.Diagnostic{
						Message: "name1: test 2",
						Location: &rdf.Location{
							Path: "cmd/reviewdog/reviewdog.go",
						},
					},
				},
			}, protocmp.Transform()); diff != "" {
				t.Errorf("%s: req.Annotation have diff:\n%s", req.Name, diff)
			}
		case "name2":
			if diff := cmp.Diff(req.Annotations, []*doghouse.Annotation{
				{
					Diagnostic: &rdf.Diagnostic{
						Message: "name2: test 1",
						Location: &rdf.Location{
							Path: "cmd/reviewdog/doghouse.go",
							Range: &rdf.Range{
								Start: &rdf.Position{Line: 14},
							},
						},
					},
				},
			}, protocmp.Transform()); diff != "" {
				t.Errorf("%s: req.Annotation have diff:\n%s", req.Name, diff)
			}
		default:
			t.Errorf("unexpected req.Name: %s", req.Name)
		}

		if err := json.NewEncoder(w).Encode(&doghouse.CheckResponse{
			ReportURL: "xxx",
		}); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := client.New(&http.Client{})
	cli.BaseURL, _ = url.Parse(ts.URL)

	// It assumes the current dir is ./cmd/reviewdog/
	var resultSet reviewdog.ResultMap
	resultSet.Store("name1", &reviewdog.Result{Diagnostics: []*rdf.Diagnostic{
		{
			Location: &rdf.Location{
				Range: &rdf.Range{Start: &rdf.Position{
					Line: 14,
				}},
				Path: "reviewdog.go", // test relative path
			},
			Message:        "name1: test 1",
			OriginalOutput: "L1\nL2",
		},
		{
			Location: &rdf.Location{
				Path: absPath(t, "reviewdog.go"), // test abs path
			},
			Message: "name1: test 2",
		},
	}})
	resultSet.Store("name2", &reviewdog.Result{Diagnostics: []*rdf.Diagnostic{
		{
			Location: &rdf.Location{
				Range: &rdf.Range{Start: &rdf.Position{
					Line: 14,
				}},
				Path: "doghouse.go",
			},
			Message: "name2: test 1",
		},
	}})

	ghInfo := &cienv.BuildInfo{
		Owner:       owner,
		Repo:        repo,
		PullRequest: prNum,
		SHA:         sha,
	}

	opt := &option{filterMode: filter.ModeAdded}
	if err := postResultSet(context.Background(), &resultSet, ghInfo, cli, opt); err != nil {
		t.Fatal(err)
	}
}

func TestPostResultSet_conclusion(t *testing.T) {
	const (
		owner = "haya14busa"
		repo  = "reviewdog"
		prNum = 14
		sha   = "1414"
	)

	tests := []struct {
		conclusion  string
		failOnError bool
		wantErr     bool
	}{
		{conclusion: "failure", failOnError: true, wantErr: true},
		{conclusion: "neutral", failOnError: true, wantErr: false},
		{conclusion: "success", failOnError: true, wantErr: false},
		{conclusion: "", failOnError: true, wantErr: false},
		{conclusion: "failure", failOnError: false, wantErr: false},
	}

	for _, tt := range tests {
		id := fmt.Sprintf("[conclusion=%s, failOnError=%v]", tt.conclusion, tt.failOnError)
		t.Run(id, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewEncoder(w).Encode(&doghouse.CheckResponse{
					ReportURL:  "xxx",
					Conclusion: tt.conclusion,
				}); err != nil {
					t.Fatal(err)
				}
				w.WriteHeader(http.StatusOK)
			})
			ts := httptest.NewServer(mux)
			defer ts.Close()

			cli := client.New(&http.Client{})
			cli.BaseURL, _ = url.Parse(ts.URL)
			var resultSet reviewdog.ResultMap
			resultSet.Store("name1", &reviewdog.Result{Diagnostics: []*rdf.Diagnostic{}})

			ghInfo := &cienv.BuildInfo{
				Owner:       owner,
				Repo:        repo,
				PullRequest: prNum,
				SHA:         sha,
			}

			opt := &option{filterMode: filter.ModeAdded, failOnError: tt.failOnError}
			err := postResultSet(context.Background(), &resultSet, ghInfo, cli, opt)
			if tt.wantErr && err == nil {
				t.Errorf("[%s] want err, but got nil.", id)
			} else if !tt.wantErr && err != nil {
				t.Errorf("[%s] got unexpected error: %v", id, err)
			}
		})
	}
}

func absPath(t *testing.T, path string) string {
	p, err := filepath.Abs(path)
	if err != nil {
		t.Errorf("filepath.Abs(%q) failed: %v", path, err)
	}
	return p
}
