package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/kylelemons/godebug/pretty"
	"golang.org/x/oauth2"

	"github.com/sezzle/reviewdog"
	"github.com/sezzle/reviewdog/filter"
	"github.com/sezzle/reviewdog/proto/rdf"
	"github.com/sezzle/reviewdog/service/commentutil"
)

const notokenSkipTestMes = "skipping test (requires actual Personal access tokens. export REVIEWDOG_TEST_GITHUB_API_TOKEN=<GitHub Personal Access Token>)"

func setupGitHubClient() *github.Client {
	token := os.Getenv("REVIEWDOG_TEST_GITHUB_API_TOKEN")
	if token == "" {
		return nil
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.TODO(), ts)
	return github.NewClient(tc)
}

func setupEnvs() (cleanup func()) {
	var cleanEnvs = []string{
		"GITHUB_ACTIONS",
	}
	saveEnvs := make(map[string]string)
	for _, key := range cleanEnvs {
		saveEnvs[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	return func() {
		for key, value := range saveEnvs {
			os.Setenv(key, value)
		}
	}
}

func moveToRootDir() {
	os.Chdir("../..")
}

func TestGitHubPullRequest_Post(t *testing.T) {
	t.Skip("skipping test which post comments actually")
	client := setupGitHubClient()
	if client == nil {
		t.Skip(notokenSkipTestMes)
	}

	// https://github.com/sezzle/reviewdog/pull/2
	owner := "haya14busa"
	repo := "reviewdog"
	pr := 2
	sha := "cce89afa9ac5519a7f5b1734db2e3aa776b138a7"

	g, err := NewGitHubPullRequest(client, owner, repo, pr, sha, "warning")
	if err != nil {
		t.Fatal(err)
	}
	comment := &reviewdog.Comment{
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: "watchdogs.go",
				},
				Message: "[reviewdog] test",
			},
			InDiffFile:    true,
			InDiffContext: true,
		},
	}
	// https://github.com/sezzle/reviewdog/pull/2/files#diff-ed1d019a10f54464cfaeaf6a736b7d27L20
	if err := g.Post(context.Background(), comment); err != nil {
		t.Error(err)
	}
	if err := g.Flush(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestGitHubPullRequest_Diff(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test which contains actual API requests in short mode")
	}
	client := setupGitHubClient()
	if client == nil {
		t.Skip(notokenSkipTestMes)
	}

	want := `diff --git a/.codecov.yml b/.codecov.yml
index aa49124774..781ee2492f 100644
--- a/.codecov.yml
+++ b/.codecov.yml
@@ -7,5 +7,4 @@ coverage:
       default:
         target: 0%
 
-comment:
-  layout: "header"
+comment: false
`

	// https://github.com/sezzle/reviewdog/pull/73
	owner := "haya14busa"
	repo := "reviewdog"
	pr := 73
	g, err := NewGitHubPullRequest(client, owner, repo, pr, "", "warning")
	if err != nil {
		t.Fatal(err)
	}
	b, err := g.Diff(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != want {
		t.Errorf("got:\n%v\nwant:\n%v", got, want)
	}
}

func TestGitHubPullRequest_comment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test which contains actual API requests in short mode")
	}
	client := setupGitHubClient()
	if client == nil {
		t.Skip(notokenSkipTestMes)
	}
	// https://github.com/sezzle/reviewdog/pull/2
	owner := "haya14busa"
	repo := "reviewdog"
	pr := 2
	g, err := NewGitHubPullRequest(client, owner, repo, pr, "", "warning")
	if err != nil {
		t.Fatal(err)
	}
	comments, err := g.comment(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range comments {
		t.Log("---")
		t.Log(*c.Body)
		t.Log(*c.Path)
		if c.Position != nil {
			t.Log(*c.Position)
		}
		t.Log(*c.CommitID)
	}
}

func TestGitHubPullRequest_Post_Flush_review_api(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	moveToRootDir()
	defer setupEnvs()()

	listCommentsAPICalled := 0
	postCommentsAPICalled := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/pulls/14/comments", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listCommentsAPICalled++
			switch r.URL.Query().Get("page") {
			default:
				cs := []*github.PullRequestComment{
					{
						Path:        github.String("reviewdog.go"),
						Line:        github.Int(2),
						Body:        github.String(commentutil.BodyPrefix + "already commented"),
						SubjectType: github.String("line"),
					},
				}
				w.Header().Add("Link", `<https://api.github.com/repos/o/r/pulls/14/comments?page=2>; rel="next"`)
				if err := json.NewEncoder(w).Encode(cs); err != nil {
					t.Fatal(err)
				}
			case "2":
				cs := []*github.PullRequestComment{
					{
						Path:        github.String("reviewdog.go"),
						Line:        github.Int(15),
						Body:        github.String(commentutil.BodyPrefix + "already commented 2"),
						SubjectType: github.String("line"),
					},
					{
						Path:        github.String("reviewdog.go"),
						StartLine:   github.Int(15),
						Line:        github.Int(16),
						Body:        github.String(commentutil.BodyPrefix + "multiline existing comment"),
						SubjectType: github.String("line"),
					},
					{
						Path:        github.String("reviewdog.go"),
						StartLine:   github.Int(15),
						Line:        github.Int(17),
						Body:        github.String(commentutil.BodyPrefix + "multiline existing comment (line-break)"),
						SubjectType: github.String("line"),
					},
					{
						Path:        github.String("reviewdog.go"),
						Line:        github.Int(1),
						Body:        github.String(commentutil.BodyPrefix + "existing file comment (no-line)"),
						SubjectType: github.String("file"),
					},
					{
						Path:        github.String("reviewdog.go"),
						Line:        github.Int(1),
						Body:        github.String(commentutil.BodyPrefix + "existing file comment (outside diff-context)"),
						SubjectType: github.String("file"),
					},
				}
				if err := json.NewEncoder(w).Encode(cs); err != nil {
					t.Fatal(err)
				}
			}
		default:
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
	})
	mux.HandleFunc("/repos/o/r/pulls/14/reviews", func(w http.ResponseWriter, r *http.Request) {
		postCommentsAPICalled++
		if r.Method != http.MethodPost {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		var req github.PullRequestReviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if *req.Event != "COMMENT" {
			t.Errorf("PullRequestReviewRequest.Event = %v, want COMMENT", *req.Event)
		}
		if req.Body != nil && *req.Body != "" {
			t.Errorf("PullRequestReviewRequest.Body = %v, want empty", *req.Body)
		}
		if *req.CommitID != "sha" {
			t.Errorf("PullRequestReviewRequest.Body = %v, want empty", *req.Body)
		}
		want := []*github.DraftReviewComment{
			{
				Path: github.String("reviewdog.go"),
				Side: github.String("RIGHT"),
				Line: github.Int(15),
				Body: github.String(commentutil.BodyPrefix + "new comment"),
			},
			{
				Path:      github.String("reviewdog.go"),
				Side:      github.String("RIGHT"),
				StartSide: github.String("RIGHT"),
				StartLine: github.Int(15),
				Line:      github.Int(16),
				Body:      github.String(commentutil.BodyPrefix + "multiline new comment"),
			},
			{
				Path:      github.String("reviewdog.go"),
				Side:      github.String("RIGHT"),
				StartSide: github.String("RIGHT"),
				StartLine: github.Int(15),
				Line:      github.Int(16),
				Body: github.String(commentutil.BodyPrefix + strings.Join([]string{
					"multiline suggestion comment",
					"```suggestion",
					"line1",
					"line2",
					"line3",
					"```",
				}, "\n") + "\n"),
			},
			{
				Path: github.String("reviewdog.go"),
				Side: github.String("RIGHT"),
				Line: github.Int(15),
				Body: github.String(commentutil.BodyPrefix + strings.Join([]string{
					"singleline suggestion comment",
					"```suggestion",
					"line1",
					"line2",
					"```",
				}, "\n") + "\n"),
			},
			{
				Path:      github.String("reviewdog.go"),
				Side:      github.String("RIGHT"),
				StartSide: github.String("RIGHT"),
				StartLine: github.Int(15),
				Line:      github.Int(16),
				Body: github.String(commentutil.BodyPrefix + strings.Join([]string{
					"invalid lines suggestion comment",
					invalidSuggestionPre + "GitHub comment range and suggestion line range must be same. L15-L16 v.s. L16-L17" + invalidSuggestionPost,
				}, "\n") + "\n"),
			},
			{
				Path:      github.String("reviewdog.go"),
				Side:      github.String("RIGHT"),
				StartSide: github.String("RIGHT"),
				StartLine: github.Int(14),
				Line:      github.Int(16),
				Body: github.String(commentutil.BodyPrefix + strings.Join([]string{
					"Use suggestion range as GitHub comment range if the suggestion is in diff context",
					"```suggestion",
					"line1",
					"line2",
					"line3",
					"```",
				}, "\n") + "\n"),
			},
			{
				Path:      github.String("reviewdog.go"),
				Side:      github.String("RIGHT"),
				StartSide: github.String("RIGHT"),
				StartLine: github.Int(14),
				Line:      github.Int(16),
				Body: github.String(commentutil.BodyPrefix + strings.Join([]string{
					"Partially invalid suggestions",
					"```suggestion",
					"line1",
					"line2",
					"line3",
					"```",
					invalidSuggestionPre + "GitHub comment range and suggestion line range must be same. L14-L16 v.s. L14-L14" + invalidSuggestionPost,
				}, "\n") + "\n"),
			},
			{
				Path:      github.String("reviewdog.go"),
				Side:      github.String("RIGHT"),
				StartSide: github.String("RIGHT"),
				StartLine: github.Int(15),
				Line:      github.Int(16),
				Body: github.String(commentutil.BodyPrefix + strings.Join([]string{
					"non-line based suggestion comment (no source lines)",
					invalidSuggestionPre + "source lines are not available" + invalidSuggestionPost,
				}, "\n") + "\n"),
			},
			{
				Path: github.String("reviewdog.go"),
				Side: github.String("RIGHT"),
				Line: github.Int(15),
				Body: github.String(commentutil.BodyPrefix + strings.Join([]string{
					"range suggestion (single line)",
					"```suggestion",
					"haya14busa",
					"```",
				}, "\n") + "\n"),
			},
			{
				Path:      github.String("reviewdog.go"),
				Side:      github.String("RIGHT"),
				StartSide: github.String("RIGHT"),
				StartLine: github.Int(15),
				Line:      github.Int(16),
				Body: github.String(commentutil.BodyPrefix + strings.Join([]string{
					"range suggestion (multi-line)",
					"```suggestion",
					"haya14busa (multi-line)",
					"```",
				}, "\n") + "\n"),
			},
			{
				Path:      github.String("reviewdog.go"),
				Side:      github.String("RIGHT"),
				StartSide: github.String("RIGHT"),
				StartLine: github.Int(15),
				Line:      github.Int(17),
				Body: github.String(commentutil.BodyPrefix + strings.Join([]string{
					"range suggestion (line-break, remove)",
					"```suggestion",
					"line 15 (content at line 15)",
					"```",
				}, "\n") + "\n"),
			},
			{
				Path: github.String("reviewdog.go"),
				Side: github.String("RIGHT"),
				Line: github.Int(15),
				Body: github.String(commentutil.BodyPrefix + strings.Join([]string{
					"range suggestion (insert)",
					"```suggestion",
					"haya14busa",
					"```",
				}, "\n") + "\n"),
			},
			{
				Path: github.String("reviewdog.go"),
				Side: github.String("RIGHT"),
				Line: github.Int(15),
				Body: github.String(commentutil.BodyPrefix + strings.Join([]string{
					"multiple suggestions",
					"```suggestion",
					"haya1busa",
					"```",
					"```suggestion",
					"haya4busa",
					"```",
					"```suggestion",
					"haya14busa",
					"```",
				}, "\n") + "\n"),
			},
			{
				Path: github.String("reviewdog.go"),
				Side: github.String("RIGHT"),
				Line: github.Int(15),
				Body: github.String(commentutil.BodyPrefix + strings.Join([]string{
					"range suggestion with start only location",
					"```suggestion",
					"haya14busa",
					"```",
				}, "\n") + "\n"),
			},
			{
				Path:      github.String("reviewdog.go"),
				Side:      github.String("RIGHT"),
				StartSide: github.String("RIGHT"),
				StartLine: github.Int(15),
				Line:      github.Int(16),
				Body: github.String(commentutil.BodyPrefix + strings.Join([]string{
					"multiline suggestion comment including a code fence block",
					"````suggestion",
					"```",
					"some code",
					"```",
					"````",
				}, "\n") + "\n"),
			},
			{
				Path: github.String("reviewdog.go"),
				Side: github.String("RIGHT"),
				Line: github.Int(15),
				Body: github.String(commentutil.BodyPrefix + strings.Join([]string{
					"singleline suggestion comment including a code fence block",
					"````suggestion",
					"```",
					"some code",
					"```",
					"````",
				}, "\n") + "\n"),
			},
			{
				Path:      github.String("reviewdog.go"),
				Side:      github.String("RIGHT"),
				StartSide: github.String("RIGHT"),
				StartLine: github.Int(15),
				Line:      github.Int(16),
				Body: github.String(commentutil.BodyPrefix + strings.Join([]string{
					"multiline suggestion comment including an empty code fence block",
					"``````suggestion",
					"```",
					"`````",
					"``````",
				}, "\n") + "\n"),
			},
		}
		if diff := pretty.Compare(want, req.Comments); diff != "" {
			t.Errorf("req.Comments diff: (-got +want)\n%s", diff)
		}
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")
	g, err := NewGitHubPullRequest(cli, "o", "r", 14, "sha", "warning")
	if err != nil {
		t.Fatal(err)
	}
	comments := []*reviewdog.Comment{
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 2,
							},
						},
					},
					Message: "already commented",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 15,
							},
						},
					},
					Message: "already commented 2",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 15,
							},
						},
					},
					Message: "new comment",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 15,
							},
							End: &rdf.Position{
								Line: 16,
							},
						},
					},
					Message: "multiline existing comment",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line:   15,
								Column: 1,
							},
							End: &rdf.Position{
								Line:   17,
								Column: 1,
							},
						},
					},
					Message: "multiline existing comment (line-break)",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 15,
							},
							End: &rdf.Position{
								Line: 16,
							},
						},
					},
					Message: "multiline new comment",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						// No Line
					},
					Message: "should not be reported via GitHub Review API",
				},
				InDiffFile:    false,
				InDiffContext: false,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						// No Line
					},
					Message: "file comment (no-line)",
				},
				InDiffFile:    true,
				InDiffContext: false,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						// No Line
					},
					Message: "existing file comment (no-line)",
				},
				InDiffFile:    true,
				InDiffContext: false,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 18,
							},
						},
					},
					Message: "file comment (outside diff-context)",
				},
				InDiffFile:    true,
				InDiffContext: false,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 18,
							},
						},
					},
					Message: "existing file comment (outside diff-context)",
				},
				InDiffFile:    true,
				InDiffContext: false,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 15,
							},
							End: &rdf.Position{
								Line: 16,
							},
						},
					},
					Suggestions: []*rdf.Suggestion{
						{
							Range: &rdf.Range{
								Start: &rdf.Position{
									Line: 15,
								},
								End: &rdf.Position{
									Line: 16,
								},
							},
							Text: "line1\nline2\nline3",
						},
					},
					Message: "multiline suggestion comment",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 15,
							},
						},
					},
					Suggestions: []*rdf.Suggestion{
						{
							Range: &rdf.Range{
								Start: &rdf.Position{
									Line: 15,
								},
							},
							Text: "line1\nline2",
						},
					},
					Message: "singleline suggestion comment",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 15,
							},
							End: &rdf.Position{
								Line: 16,
							},
						},
					},
					Suggestions: []*rdf.Suggestion{
						{
							Range: &rdf.Range{
								Start: &rdf.Position{
									Line: 16,
								},
								End: &rdf.Position{
									Line: 17,
								},
							},
							Text: "line1\nline2\nline3",
						},
					},
					Message: "invalid lines suggestion comment",
				},
				InDiffFile:                   true,
				InDiffContext:                true,
				FirstSuggestionInDiffContext: false,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				SourceLines: map[int]string{
					14: "line 14 before",
					15: "line 15 before",
					16: "line 16 before",
				},
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 15,
							},
						},
					},
					Suggestions: []*rdf.Suggestion{
						{
							Range: &rdf.Range{
								Start: &rdf.Position{
									Line: 14,
								},
								End: &rdf.Position{
									Line: 16,
								},
							},
							Text: "line1\nline2\nline3",
						},
					},
					Message: "Use suggestion range as GitHub comment range if the suggestion is in diff context",
				},
				InDiffFile:                   true,
				InDiffContext:                true,
				FirstSuggestionInDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				SourceLines: map[int]string{
					14: "line 14 before",
					15: "line 15 before",
					16: "line 16 before",
				},
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 15,
							},
						},
					},
					Suggestions: []*rdf.Suggestion{
						{
							Range: &rdf.Range{
								Start: &rdf.Position{
									Line: 14,
								},
								End: &rdf.Position{
									Line: 16,
								},
							},
							Text: "line1\nline2\nline3",
						},
						{
							Range: &rdf.Range{
								Start: &rdf.Position{
									Line: 14,
								},
								End: &rdf.Position{
									Line: 14,
								},
							},
							Text: "line1\nline2",
						},
					},
					Message: "Partially invalid suggestions",
				},
				InDiffFile:                   true,
				InDiffContext:                true,
				FirstSuggestionInDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 15,
							},
							End: &rdf.Position{
								Line: 16,
							},
						},
					},
					Suggestions: []*rdf.Suggestion{
						{
							Range: &rdf.Range{
								Start: &rdf.Position{
									Line:   15,
									Column: 5,
								},
								End: &rdf.Position{
									Line:   16,
									Column: 7,
								},
							},
							Text: "replacement",
						},
					},
					Message: "non-line based suggestion comment (no source lines)",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				SourceLines: map[int]string{15: "haya15busa"},
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 15, Column: 5},
							End:   &rdf.Position{Line: 15, Column: 7},
						},
					},
					Suggestions: []*rdf.Suggestion{
						{
							Range: &rdf.Range{
								Start: &rdf.Position{Line: 15, Column: 5},
								End:   &rdf.Position{Line: 15, Column: 7},
							},
							Text: "14",
						},
					},
					Message: "range suggestion (single line)",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				SourceLines: map[int]string{
					15: "haya???",
					16: "???busa (multi-line)",
				},
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 15, Column: 5},
							End:   &rdf.Position{Line: 16, Column: 4},
						},
					},
					Suggestions: []*rdf.Suggestion{
						{
							Range: &rdf.Range{
								Start: &rdf.Position{Line: 15, Column: 5},
								End:   &rdf.Position{Line: 16, Column: 4},
							},
							Text: "14",
						},
					},
					Message: "range suggestion (multi-line)",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				SourceLines: map[int]string{
					15: "line 15 xxx",
					16: "line 16",
					17: "(content at line 15)",
				},
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 15, Column: 9},
							End:   &rdf.Position{Line: 17, Column: 1},
						},
					},
					Suggestions: []*rdf.Suggestion{
						{
							Range: &rdf.Range{
								Start: &rdf.Position{Line: 15, Column: 9},
								End:   &rdf.Position{Line: 17, Column: 1},
							},
							Text: "",
						},
					},
					Message: "range suggestion (line-break, remove)",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				SourceLines: map[int]string{
					15: "hayabusa",
				},
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 15, Column: 5},
							End:   &rdf.Position{Line: 15, Column: 5},
						},
					},
					Suggestions: []*rdf.Suggestion{
						{
							Range: &rdf.Range{
								Start: &rdf.Position{Line: 15, Column: 5},
								End:   &rdf.Position{Line: 15, Column: 5},
							},
							Text: "14",
						},
					},
					Message: "range suggestion (insert)",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				SourceLines: map[int]string{15: "haya??busa"},
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 15, Column: 5},
							End:   &rdf.Position{Line: 15, Column: 7},
						},
					},
					Suggestions: []*rdf.Suggestion{
						{
							Range: &rdf.Range{
								Start: &rdf.Position{Line: 15, Column: 5},
								End:   &rdf.Position{Line: 15, Column: 7},
							},
							Text: "1",
						},
						{
							Range: &rdf.Range{
								Start: &rdf.Position{Line: 15, Column: 5},
								End:   &rdf.Position{Line: 15, Column: 7},
							},
							Text: "4",
						},
						{
							Range: &rdf.Range{
								Start: &rdf.Position{Line: 15, Column: 5},
								End:   &rdf.Position{Line: 15, Column: 7},
							},
							Text: "14",
						},
					},
					Message: "multiple suggestions",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				SourceLines: map[int]string{15: "haya15busa"},
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 15, Column: 5},
						},
					},
					Suggestions: []*rdf.Suggestion{
						{
							Range: &rdf.Range{
								Start: &rdf.Position{Line: 15, Column: 5},
								End:   &rdf.Position{Line: 15, Column: 7},
							},
							Text: "14",
						},
					},
					Message: "range suggestion with start only location",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 15,
							},
							End: &rdf.Position{
								Line: 16,
							},
						},
					},
					Suggestions: []*rdf.Suggestion{
						{
							Range: &rdf.Range{
								Start: &rdf.Position{
									Line: 15,
								},
								End: &rdf.Position{
									Line: 16,
								},
							},
							Text: "```\nsome code\n```",
						},
					},
					Message: "multiline suggestion comment including a code fence block",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 15,
							},
						},
					},
					Suggestions: []*rdf.Suggestion{
						{
							Range: &rdf.Range{
								Start: &rdf.Position{
									Line: 15,
								},
							},
							Text: "```\nsome code\n```",
						},
					},
					Message: "singleline suggestion comment including a code fence block",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line: 15,
							},
							End: &rdf.Position{
								Line: 16,
							},
						},
					},
					Suggestions: []*rdf.Suggestion{
						{
							Range: &rdf.Range{
								Start: &rdf.Position{
									Line: 15,
								},
								End: &rdf.Position{
									Line: 16,
								},
							},
							Text: "```\n`````",
						},
					},
					Message: "multiline suggestion comment including an empty code fence block",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
		},
	}
	for _, c := range comments {
		if err := g.Post(context.Background(), c); err != nil {
			t.Error(err)
		}
	}
	if err := g.Flush(context.Background()); err != nil {
		t.Error(err)
	}
	if listCommentsAPICalled != 2 {
		t.Errorf("GitHub List PullRequest comments API called %v times, want 2 times", listCommentsAPICalled)
	}
	if postCommentsAPICalled != 1 {
		t.Errorf("GitHub post PullRequest comments API called %v times, want 1 times", postCommentsAPICalled)
	}
}

func TestGitHubPullRequest_Post_toomany(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	moveToRootDir()
	defer setupEnvs()()

	listCommentsAPICalled := 0
	postCommentsAPICalled := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/pulls/14/comments", func(w http.ResponseWriter, r *http.Request) {
		listCommentsAPICalled++
		if err := json.NewEncoder(w).Encode([]*github.PullRequestComment{}); err != nil {
			t.Fatal(err)
		}
	})
	mux.HandleFunc("/repos/o/r/pulls/14/reviews", func(w http.ResponseWriter, r *http.Request) {
		postCommentsAPICalled++
		var req github.PullRequestReviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if req.GetBody() == "" {
			t.Errorf("PullRequestReviewRequest.Body is empty but want some summary text")
		}
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")
	g, err := NewGitHubPullRequest(cli, "o", "r", 14, "sha", "warning")
	if err != nil {
		t.Fatal(err)
	}
	var comments []*reviewdog.Comment
	for i := 0; i < 100; i++ {
		comments = append(comments, &reviewdog.Comment{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "reviewdog.go",
						Range: &rdf.Range{Start: &rdf.Position{
							Line: int32(i),
						}},
					},
					Message: "comment",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
			ToolName: "tool",
		})
	}
	for _, c := range comments {
		if err := g.Post(context.Background(), c); err != nil {
			t.Error(err)
		}
	}
	if err := g.Flush(context.Background()); err != nil {
		t.Error(err)
	}
	if want := 1; listCommentsAPICalled != want {
		t.Errorf("GitHub List PullRequest comments API called %v times, want %d times", listCommentsAPICalled, want)
	}
	if want := 1; postCommentsAPICalled != want {
		t.Errorf("GitHub post PullRequest comments API called %v times, want %d times", postCommentsAPICalled, want)
	}
}

func TestGitHubPullRequest_workdir(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	moveToRootDir()
	defer setupEnvs()()

	g, err := NewGitHubPullRequest(nil, "", "", 0, "", "warning")
	if err != nil {
		t.Fatal(err)
	}
	if g.wd != "" {
		t.Fatalf("g.wd = %q, want empty", g.wd)
	}
	ctx := context.Background()
	want := "a/b/c"
	g.Post(ctx, &reviewdog.Comment{Result: &filter.FilteredDiagnostic{
		Diagnostic: &rdf.Diagnostic{Location: &rdf.Location{Path: want}}}})
	if got := g.postComments[0].Result.Diagnostic.GetLocation().GetPath(); got != want {
		t.Errorf("wd=%q path=%q, want %q", g.wd, got, want)
	}

	subDir := "cmd/"
	if err := os.Chdir(subDir); err != nil {
		t.Fatal(err)
	}
	g, _ = NewGitHubPullRequest(nil, "", "", 0, "", "warning")
	if g.wd != subDir {
		t.Fatalf("gitRelWorkdir() = %q, want %q", g.wd, subDir)
	}
	path := "a/b/c"
	wantPath := "cmd/" + path
	g.Post(ctx, &reviewdog.Comment{Result: &filter.FilteredDiagnostic{
		Diagnostic: &rdf.Diagnostic{Location: &rdf.Location{Path: want}}}})
	if got := g.postComments[0].Result.Diagnostic.GetLocation().GetPath(); got != wantPath {
		t.Errorf("wd=%q path=%q, want %q", g.wd, got, wantPath)
	}
}

func TestGitHubPullRequest_Diff_fake(t *testing.T) {
	apiCalled := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/pulls/14", func(w http.ResponseWriter, r *http.Request) {
		apiCalled++
		if r.Method != http.MethodGet {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		if accept := r.Header.Get("Accept"); !strings.Contains(accept, "diff") {
			t.Errorf("Accept header doesn't contain 'diff': %v", accept)
		}
		w.Write([]byte("Pull Request diff"))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")
	g, err := NewGitHubPullRequest(cli, "o", "r", 14, "sha", "warning")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Diff(context.Background()); err != nil {
		t.Fatal(err)
	}
	if apiCalled != 1 {
		t.Errorf("GitHub API should be called once; called %v times", apiCalled)
	}
}

func TestGitHubPullRequest_Diff_fake_fallback(t *testing.T) {
	apiCalled := 0
	mux := http.NewServeMux()
	headSHA := "HEAD^"
	baseSHA := "HEAD"
	mux.HandleFunc("/repos/o/r/pulls/14", func(w http.ResponseWriter, r *http.Request) {
		apiCalled++
		if r.Method != http.MethodGet {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		if accept := r.Header.Get("Accept"); strings.Contains(accept, "diff") {
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}
		if accept := r.Header.Get("Accept"); accept != "application/vnd.github.v3+json" {
			t.Errorf("Accept header doesn't contain 'diff': %v", accept)
		}

		pullRequestJSON, err := json.Marshal(github.PullRequest{
			Head: &github.PullRequestBranch{
				SHA: &headSHA,
			},
			Base: &github.PullRequestBranch{
				SHA: &baseSHA,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := w.Write(pullRequestJSON); err != nil {
			t.Fatal(err)
		}
	})
	mux.HandleFunc("/repos/o/r/compare/"+headSHA+"..."+baseSHA, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		if accept := r.Header.Get("Accept"); accept != "application/vnd.github.v3+json" {
			t.Errorf("Accept header doesn't contain 'diff': %v", accept)
		}

		mergeBaseSha := "HEAD^"

		commitsComparisonJSON, err := json.Marshal(github.CommitsComparison{
			MergeBaseCommit: &github.RepositoryCommit{
				SHA: &mergeBaseSha,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := w.Write(commitsComparisonJSON); err != nil {
			t.Fatal(err)
		}
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Setenv("REVIEWDOG_SKIP_GIT_FETCH", "true")

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")
	g, err := NewGitHubPullRequest(cli, "o", "r", 14, "sha", "warning")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Diff(context.Background()); err != nil {
		t.Fatal(err)
	}
	if apiCalled != 2 {
		t.Errorf("GitHub API should be called twice; called %v times", apiCalled)
	}
}

func TestGitHubPullRequest_Post_NoPermission(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	moveToRootDir()
	t.Setenv("GITHUB_ACTIONS", "true")

	listCommentsAPICalled := 0
	postCommentsAPICalled := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/pulls/14/comments", func(w http.ResponseWriter, r *http.Request) {
		listCommentsAPICalled++
		if err := json.NewEncoder(w).Encode([]*github.PullRequestComment{}); err != nil {
			t.Fatal(err)
		}
	})
	mux.HandleFunc("/repos/o/r/pulls/14/reviews", func(w http.ResponseWriter, r *http.Request) {
		postCommentsAPICalled++
		w.WriteHeader(http.StatusNotFound)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")
	g, err := NewGitHubPullRequest(cli, "o", "r", 14, "sha", "warning")
	if err != nil {
		t.Fatal(err)
	}
	comments := []*reviewdog.Comment{
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "service/github/github_test.go",
						Range: &rdf.Range{Start: &rdf.Position{
							Line: 1,
						}},
					},
					Message: "test message for TestGitHubPullRequest_Post_NoPermission",
				},
				InDiffFile:    true,
				InDiffContext: true,
			},
			ToolName: "service/github/github_test.go",
		},
	}
	for _, c := range comments {
		if err := g.Post(context.Background(), c); err != nil {
			t.Error(err)
		}
	}
	if err := g.Flush(context.Background()); err != nil {
		t.Error(err)
	}
	if want := 1; listCommentsAPICalled != want {
		t.Errorf("GitHub List PullRequest comments API called %v times, want %d times", listCommentsAPICalled, want)
	}
	if want := 1; postCommentsAPICalled != want {
		t.Errorf("GitHub post PullRequest comments API called %v times, want %d times", postCommentsAPICalled, want)
	}
}
