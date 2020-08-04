package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/cienv"
	"github.com/reviewdog/reviewdog/doghouse"
	"github.com/reviewdog/reviewdog/doghouse/client"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/project"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

func setupEnvs(testEnvs map[string]string) (cleanup func()) {
	saveEnvs := make(map[string]string)
	for key, value := range testEnvs {
		saveEnvs[key] = os.Getenv(key)
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}
	return func() {
		for key, value := range saveEnvs {
			os.Setenv(key, value)
		}
	}
}

func TestNewDoghouseCli_returnGitHubClient(t *testing.T) {
	cleanup := setupEnvs(map[string]string{
		"REVIEWDOG_TOKEN":            "",
		"GITHUB_ACTIONS":             "xxx",
		"REVIEWDOG_GITHUB_API_TOKEN": "xxx",
	})
	defer cleanup()
	cli, err := newDoghouseCli(context.Background())
	if err != nil {
		t.Fatalf("failed to create new client: %v", err)
	}
	if _, ok := cli.(*client.GitHubClient); !ok {
		t.Errorf("got %T client, want *client.GitHubClient client", cli)
	}
}

func TestNewDoghouseCli_returnErrorForGitHubClient(t *testing.T) {
	cleanup := setupEnvs(map[string]string{
		"REVIEWDOG_TOKEN":            "",
		"GITHUB_ACTIONS":             "true",
		"REVIEWDOG_GITHUB_API_TOKEN": "", // missing
	})
	defer cleanup()
	if _, err := newDoghouseCli(context.Background()); err == nil {
		t.Error("got no error but want REVIEWDOG_GITHUB_API_TOKEN missing error")
	}
}

func TestNewDoghouseCli_returnDogHouseClientWithReviewdogToken(t *testing.T) {
	cleanup := setupEnvs(map[string]string{
		"REVIEWDOG_TOKEN":            "xxx",
		"GITHUB_ACTIONS":             "true",
		"REVIEWDOG_GITHUB_API_TOKEN": "xxx",
	})
	defer cleanup()
	cli, err := newDoghouseCli(context.Background())
	if err != nil {
		t.Fatalf("failed to create new client: %v", err)
	}
	if _, ok := cli.(*client.DogHouseClient); !ok {
		t.Errorf("got %T client, want *client.DogHouseClient client", cli)
	}
}

func TestNewDoghouseCli_returnDogHouseClient(t *testing.T) {
	cleanup := setupEnvs(map[string]string{
		"REVIEWDOG_TOKEN":            "",
		"GITHUB_ACTIONS":             "",
		"REVIEWDOG_GITHUB_API_TOKEN": "",
	})
	defer cleanup()
	cli, err := newDoghouseCli(context.Background())
	if err != nil {
		t.Fatalf("failed to create new client: %v", err)
	}
	if _, ok := cli.(*client.DogHouseClient); !ok {
		t.Errorf("got %T client, want *client.DogHouseClient client", cli)
	}
}

func TestNewDoghouseServerCli(t *testing.T) {
	if _, ok := newDoghouseServerCli(context.Background()).Client.Transport.(*oauth2.Transport); ok {
		t.Error("got oauth2 http client, want default client")
	}

	cleanup := setupEnvs(map[string]string{
		"REVIEWDOG_TOKEN": "xxx",
	})
	defer cleanup()

	if _, ok := newDoghouseServerCli(context.Background()).Client.Transport.(*oauth2.Transport); !ok {
		t.Error("w/ TOKEN: got unexpected http client, want oauth client")
	}
}

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

	tmp, err := ioutil.TempFile("", "")
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

type fakeDoghouseServerCli struct {
	client.DogHouseClientInterface
	FakeCheck func(context.Context, *doghouse.CheckRequest) (*doghouse.CheckResponse, error)
}

func (f *fakeDoghouseServerCli) Check(ctx context.Context, req *doghouse.CheckRequest) (*doghouse.CheckResponse, error) {
	return f.FakeCheck(ctx, req)
}

func TestPostResultSet_withReportURL(t *testing.T) {
	const (
		owner = "haya14busa"
		repo  = "reviewdog"
		prNum = 14
		sha   = "1414"
	)

	fakeCli := &fakeDoghouseServerCli{}
	fakeCli.FakeCheck = func(ctx context.Context, req *doghouse.CheckRequest) (*doghouse.CheckResponse, error) {
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
		return &doghouse.CheckResponse{ReportURL: "xxx"}, nil
	}

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
	if _, err := postResultSet(context.Background(), &resultSet, ghInfo, fakeCli, opt); err != nil {
		t.Fatal(err)
	}
}

func TestPostResultSet_withoutReportURL(t *testing.T) {
	const (
		owner = "haya14busa"
		repo  = "reviewdog"
		prNum = 14
		sha   = "1414"
	)

	wantResults := []*filter.FilteredDiagnostic{{ShouldReport: true}}
	fakeCli := &fakeDoghouseServerCli{}
	fakeCli.FakeCheck = func(ctx context.Context, req *doghouse.CheckRequest) (*doghouse.CheckResponse, error) {
		return &doghouse.CheckResponse{CheckedResults: wantResults}, nil
	}

	var resultSet reviewdog.ResultMap
	resultSet.Store("name1", &reviewdog.Result{Diagnostics: []*rdf.Diagnostic{}})

	ghInfo := &cienv.BuildInfo{Owner: owner, Repo: repo, PullRequest: prNum, SHA: sha}

	opt := &option{filterMode: filter.ModeAdded}
	resp, err := postResultSet(context.Background(), &resultSet, ghInfo, fakeCli, opt)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Len() == 0 {
		t.Fatal("result should not be empty")
	}
	results, err := resp.Load("name1")
	if err != nil {
		t.Fatalf("should have result for name1: %v", err)
	}
	if diff := cmp.Diff(results.FilteredDiagnostic, wantResults, protocmp.Transform()); diff != "" {
		t.Errorf("results has diff:\n%s", diff)
	}
}

func TestPostResultSet_conclusion(t *testing.T) {
	const (
		owner = "haya14busa"
		repo  = "reviewdog"
		prNum = 14
		sha   = "1414"
	)

	fakeCli := &fakeDoghouseServerCli{}
	var resultSet reviewdog.ResultMap
	resultSet.Store("name1", &reviewdog.Result{Diagnostics: []*rdf.Diagnostic{}})
	ghInfo := &cienv.BuildInfo{Owner: owner, Repo: repo, PullRequest: prNum, SHA: sha}

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
		tt := tt
		fakeCli.FakeCheck = func(ctx context.Context, req *doghouse.CheckRequest) (*doghouse.CheckResponse, error) {
			return &doghouse.CheckResponse{ReportURL: "xxx", Conclusion: tt.conclusion}, nil
		}
		opt := &option{filterMode: filter.ModeAdded, failOnError: tt.failOnError}
		id := fmt.Sprintf("[conclusion=%s, failOnError=%v]", tt.conclusion, tt.failOnError)
		_, err := postResultSet(context.Background(), &resultSet, ghInfo, fakeCli, opt)
		if tt.wantErr && err == nil {
			t.Errorf("[%s] want err, but got nil.", id)
		} else if !tt.wantErr && err != nil {
			t.Errorf("[%s] got unexpected error: %v", id, err)
		}
	}
}

func TestPostResultSet_withEmptyResponse(t *testing.T) {
	const (
		owner = "haya14busa"
		repo  = "reviewdog"
		prNum = 14
		sha   = "1414"
	)

	fakeCli := &fakeDoghouseServerCli{}
	fakeCli.FakeCheck = func(ctx context.Context, req *doghouse.CheckRequest) (*doghouse.CheckResponse, error) {
		return &doghouse.CheckResponse{}, nil
	}

	var resultSet reviewdog.ResultMap
	resultSet.Store("name1", &reviewdog.Result{Diagnostics: []*rdf.Diagnostic{}})

	ghInfo := &cienv.BuildInfo{Owner: owner, Repo: repo, PullRequest: prNum, SHA: sha}

	opt := &option{filterMode: filter.ModeAdded}
	if _, err := postResultSet(context.Background(), &resultSet, ghInfo, fakeCli, opt); err == nil {
		t.Error("got no error but want report missing error")
	}
}

func TestReportResults(t *testing.T) {
	cleanup := setupEnvs(map[string]string{
		"GITHUB_ACTIONS":    "",
		"GITHUB_EVENT_PATH": "",
	})
	defer cleanup()
	filteredResultSet := new(reviewdog.FilteredResultMap)
	filteredResultSet.Store("name1", &reviewdog.FilteredResult{
		FilteredDiagnostic: []*filter.FilteredDiagnostic{
			{
				Diagnostic: &rdf.Diagnostic{
					OriginalOutput: "name1-L1\nname1-L2",
				},
				ShouldReport: true,
			},
			{
				Diagnostic: &rdf.Diagnostic{
					OriginalOutput: "name1.2-L1\nname1.2-L2",
				},
				ShouldReport: false,
			},
		},
	})
	filteredResultSet.Store("name2", &reviewdog.FilteredResult{
		FilteredDiagnostic: []*filter.FilteredDiagnostic{
			{
				Diagnostic: &rdf.Diagnostic{
					OriginalOutput: "name1-L1\nname1-L2",
				},
				ShouldReport: false,
			},
		},
	})
	stdout := new(bytes.Buffer)
	foundResultShouldReport := reportResults(stdout, filteredResultSet)
	if !foundResultShouldReport {
		t.Errorf("foundResultShouldReport = %v, want true", foundResultShouldReport)
	}
	want := `reviewdog: Reporting results for "name1"
name1-L1
name1-L2
reviewdog: Reporting results for "name2"
reviewdog: No results found for "name2". 1 results found outside diff.
`
	if got := stdout.String(); got != want {
		t.Errorf("diff found for report:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestReportResults_inGitHubAction(t *testing.T) {
	cleanup := setupEnvs(map[string]string{
		"GITHUB_ACTIONS":    "true",
		"GITHUB_EVENT_PATH": "",
	})
	defer cleanup()
	filteredResultSet := new(reviewdog.FilteredResultMap)
	filteredResultSet.Store("name1", &reviewdog.FilteredResult{
		FilteredDiagnostic: []*filter.FilteredDiagnostic{
			{
				Diagnostic: &rdf.Diagnostic{
					OriginalOutput: "name1-L1\nname1-L2",
				},
				ShouldReport: true,
			},
		},
	})
	stdout := new(bytes.Buffer)
	_ = reportResults(stdout, filteredResultSet)
	want := `reviewdog: Reporting results for "name1"
`
	if got := stdout.String(); got != want {
		t.Errorf("diff found for report:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestReportResults_noResultsShouldReport(t *testing.T) {
	cleanup := setupEnvs(map[string]string{
		"GITHUB_ACTIONS":    "",
		"GITHUB_EVENT_PATH": "",
	})
	defer cleanup()
	filteredResultSet := new(reviewdog.FilteredResultMap)
	filteredResultSet.Store("name1", &reviewdog.FilteredResult{
		FilteredDiagnostic: []*filter.FilteredDiagnostic{
			{
				Diagnostic: &rdf.Diagnostic{
					OriginalOutput: "name1-L1\nname1-L2",
				},
				ShouldReport: false,
			},
			{
				Diagnostic: &rdf.Diagnostic{
					OriginalOutput: "name1.2-L1\nname1.2-L2",
				},
				ShouldReport: false,
			},
		},
	})
	filteredResultSet.Store("name2", &reviewdog.FilteredResult{
		FilteredDiagnostic: []*filter.FilteredDiagnostic{
			{
				Diagnostic: &rdf.Diagnostic{
					OriginalOutput: "name1-L1\nname1-L2",
				},
				ShouldReport: false,
			},
		},
	})
	stdout := new(bytes.Buffer)
	foundResultShouldReport := reportResults(stdout, filteredResultSet)
	if foundResultShouldReport {
		t.Errorf("foundResultShouldReport = %v, want false", foundResultShouldReport)
	}
	want := `reviewdog: Reporting results for "name1"
reviewdog: No results found for "name1". 2 results found outside diff.
reviewdog: Reporting results for "name2"
reviewdog: No results found for "name2". 1 results found outside diff.
`
	if got := stdout.String(); got != want {
		t.Errorf("diff found for report:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func absPath(t *testing.T, path string) string {
	p, err := filepath.Abs(path)
	if err != nil {
		t.Errorf("filepath.Abs(%q) failed: %v", path, err)
	}
	return p
}
