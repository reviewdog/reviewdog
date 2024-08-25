package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-github/v64/github"
	"github.com/kylelemons/godebug/pretty"
	"golang.org/x/oauth2"

	"github.com/google/go-cmp/cmp"
	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
	"github.com/reviewdog/reviewdog/service/commentutil"
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

	// https://github.com/reviewdog/reviewdog/pull/2
	owner := "haya14busa"
	repo := "reviewdog"
	pr := 2
	sha := "cce89afa9ac5519a7f5b1734db2e3aa776b138a7"

	g, err := NewGitHubPullRequest(client, owner, repo, pr, sha, "warning", "tool-name")
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
	// https://github.com/reviewdog/reviewdog/pull/2/files#diff-ed1d019a10f54464cfaeaf6a736b7d27L20
	if err := g.Post(context.Background(), comment); err != nil {
		t.Error(err)
	}
	if err := g.Flush(context.Background()); err != nil {
		t.Error(err)
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
	// https://github.com/reviewdog/reviewdog/pull/2
	owner := "haya14busa"
	repo := "reviewdog"
	pr := 2
	g, err := NewGitHubPullRequest(client, owner, repo, pr, "", "warning", "tool-name")
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
	postReviewCommentAPICalled := 0
	postPullRequestCommentAPICalled := 0
	repoAPICalled := 0
	delCommentsAPICalled := 0
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
						Body:        github.String(commentutil.BodyPrefix + "already commented" + "\n<!-- __reviewdog__:ChBmMzg0YTRlZDRkYTViOTZl -->\n"),
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
						Body:        github.String(commentutil.BodyPrefix + "already commented 2" + "\n<!-- __reviewdog__:ChAxNDgzY2EyNTY0MjU2NmYx -->\n"),
						SubjectType: github.String("line"),
					},
					{
						ID:          github.Int64(1414),
						Path:        github.String("reviewdog.go"),
						Line:        github.Int(15),
						Body:        github.String(commentutil.BodyPrefix + "already commented [outdated]" + "\n<!-- __reviewdog__:Cg9jY2FlN2NlYTg0M2M0MDISCXRvb2wtbmFtZQ== -->\n"),
						SubjectType: github.String("line"),
					},
					{
						ID:          github.Int64(1414),
						Path:        github.String("reviewdog.go"),
						Line:        github.Int(15),
						Body:        github.String(commentutil.BodyPrefix + "already commented [different tool]" + "\n<!-- __reviewdog__:CgZ4eHh4eHgSDmRpZmZlcmVudC10b29s -->\n"),
						SubjectType: github.String("line"),
					},
					{
						Path:        github.String("reviewdog.go"),
						StartLine:   github.Int(15),
						Line:        github.Int(16),
						Body:        github.String(commentutil.BodyPrefix + "multiline existing comment" + "\n<!-- __reviewdog__:ChBjNGNiNTRjMDc2YjNhMjcx -->\n"),
						SubjectType: github.String("line"),
					},
					{
						Path:        github.String("reviewdog.go"),
						StartLine:   github.Int(15),
						Line:        github.Int(17),
						Body:        github.String(commentutil.BodyPrefix + "multiline existing comment (line-break)" + "\n<!-- __reviewdog__:ChA2NjI1ZDI2MGJmNTdhNjUw -->\n"),
						SubjectType: github.String("line"),
					},
					{
						Path:        github.String("reviewdog.go"),
						Line:        github.Int(1),
						Body:        github.String(commentutil.BodyPrefix + "existing file comment (no-line)" + "\n<!-- __reviewdog__:ChA2ZDI2MGNmYjY3NTQ4YTgxEgl0b29sLW5hbWU= -->\n"),
						SubjectType: github.String("file"),
					},
					{
						Path:        github.String("reviewdog.go"),
						Line:        github.Int(1),
						Body:        github.String(commentutil.BodyPrefix + "existing file comment (outside diff-context)" + "\n<!-- __reviewdog__:ChAyMzFjY2Q1ZWRhMjRkM2ZhEgl0b29sLW5hbWU= -->\n"),
						SubjectType: github.String("file"),
					},
				}
				if err := json.NewEncoder(w).Encode(cs); err != nil {
					t.Fatal(err)
				}
			}
		case http.MethodPost:
			postPullRequestCommentAPICalled++
			var req github.PullRequestComment
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error(err)
			}
			expects := []github.PullRequestComment{
				{
					Body:        github.String("<sub>reported by [reviewdog](https://github.com/reviewdog/reviewdog) :dog:</sub><br>file comment (no-line)\n<!-- __reviewdog__:ChBkZDlkMDllNmM5MTllODU1Egl0b29sLW5hbWU= -->\n"),
					Path:        github.String("reviewdog.go"),
					Side:        github.String("RIGHT"),
					CommitID:    github.String("sha"),
					SubjectType: github.String("file"),
				},
				{
					Body: github.String(`<sub>reported by [reviewdog](https://github.com/reviewdog/reviewdog) :dog:</sub><br>file comment (outside diff-context)

https://test/repo/path/blob/sha/reviewdog.go#L18
<!-- __reviewdog__:ChA5Mzc1OWY5ZTRmMmI5NThhEgl0b29sLW5hbWU= -->
`),
					Path:        github.String("reviewdog.go"),
					Side:        github.String("RIGHT"),
					CommitID:    github.String("sha"),
					SubjectType: github.String("file"),
				},
				{},
			}
			want := expects[postPullRequestCommentAPICalled-1]
			if diff := cmp.Diff(req, want); diff != "" {
				t.Errorf("result has diff (API call: %d):\n%s", postPullRequestCommentAPICalled, diff)
			}
		default:
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
	})
	mux.HandleFunc("/repos/o/r/pulls/14/reviews", func(w http.ResponseWriter, r *http.Request) {
		postReviewCommentAPICalled++
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
				Body: github.String(commentutil.BodyPrefix + "new comment" + "\n<!-- __reviewdog__:xxxxxxxxxx -->\n"),
			},
			{
				Path:      github.String("reviewdog.go"),
				Side:      github.String("RIGHT"),
				StartSide: github.String("RIGHT"),
				StartLine: github.Int(15),
				Line:      github.Int(16),
				Body:      github.String(commentutil.BodyPrefix + "multiline new comment" + "\n<!-- __reviewdog__:xxxxxxxxxx -->\n"),
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
					"",
					"<!-- __reviewdog__:xxxxxxxxxx -->",
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
					"",
					"<!-- __reviewdog__:xxxxxxxxxx -->",
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
					"",
					"<!-- __reviewdog__:xxxxxxxxxx -->",
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
					"",
					"<!-- __reviewdog__:xxxxxxxxxx -->",
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
					"",
					"<!-- __reviewdog__:xxxxxxxxxx -->",
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
					"",
					"<!-- __reviewdog__:xxxxxxxxxx -->",
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
					"",
					"<!-- __reviewdog__:xxxxxxxxxx -->",
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
					"",
					"<!-- __reviewdog__:xxxxxxxxxx -->",
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
					"",
					"<!-- __reviewdog__:xxxxxxxxxx -->",
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
					"",
					"<!-- __reviewdog__:xxxxxxxxxx -->",
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
					"",
					"<!-- __reviewdog__:xxxxxxxxxx -->",
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
					"",
					"<!-- __reviewdog__:xxxxxxxxxx -->",
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
					"",
					"<!-- __reviewdog__:xxxxxxxxxx -->",
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
					"",
					"<!-- __reviewdog__:xxxxxxxxxx -->",
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
					"",
					"<!-- __reviewdog__:xxxxxxxxxx -->",
				}, "\n") + "\n"),
			},
			{
				Path:      github.String("reviewdog.go"),
				Side:      github.String("RIGHT"),
				StartSide: github.String("RIGHT"),
				StartLine: github.Int(15),
				Line:      github.Int(16),
				Body: github.String(commentutil.BodyPrefix + strings.Join([]string{
					"related location test",
					"<hr>",
					"",
					"related loc test",
					"https://test/repo/path/blob/sha/reviewdog.go#L14-L16",
					"<hr>",
					"",
					"related loc test (2)",
					"https://test/repo/path/blob/sha/service/github/reviewdog2.go#L14",
					"<!-- __reviewdog__:xxxxxxxxxx -->",
					"",
				}, "\n")),
			},
		}
		// Replace __reviewdog__ comment so that the test pass regardless of environments.
		// Proto serialization is not canonical, and test could break unless
		// replacing the metacomment string.
		for i := 0; i < len(req.Comments); i++ {
			metaCommentRe := regexp.MustCompile(`__reviewdog__:\S+`)
			req.Comments[i].Body = github.String(metaCommentRe.ReplaceAllString(*req.Comments[i].Body, `__reviewdog__:xxxxxxxxxx`))
		}
		if diff := pretty.Compare(want, req.Comments); diff != "" {
			t.Errorf("req.Comments diff: (-got +want)\n%s", diff)
		}
	})
	mux.HandleFunc("/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		repoAPICalled++
		if err := json.NewEncoder(w).Encode(&github.Repository{
			HTMLURL: github.String("https://test/repo/path"),
		}); err != nil {
			t.Fatal(err)
		}
	})
	mux.HandleFunc("/repos/o/r/pulls/comments/1414", func(w http.ResponseWriter, r *http.Request) {
		delCommentsAPICalled++
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")
	g, err := NewGitHubPullRequest(cli, "o", "r", 14, "sha", "warning", "tool-name")
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
					Message: "related location test",
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
	if postReviewCommentAPICalled != 1 {
		t.Errorf("GitHub post Review comments API called %v times, want 1 times", postReviewCommentAPICalled)
	}
	if postPullRequestCommentAPICalled != 2 {
		t.Errorf("GitHub post PullRequest comments API called %v times, want 2 times", postPullRequestCommentAPICalled)
	}
	if repoAPICalled != 1 {
		t.Errorf("GitHub Repository API called %v times, want 1 times", repoAPICalled)
	}
	if delCommentsAPICalled != 1 {
		t.Errorf("GitHub Delete PullRequest comments API called %v times, want 1 times", delCommentsAPICalled)
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
	mux.HandleFunc("/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(&github.Repository{
			HTMLURL: github.String("https://test/repo/path"),
		}); err != nil {
			t.Fatal(err)
		}
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")
	g, err := NewGitHubPullRequest(cli, "o", "r", 14, "sha", "warning", "tool-name")
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

	g, err := NewGitHubPullRequest(nil, "", "", 0, "", "warning", "tool-name")
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
	g, _ = NewGitHubPullRequest(nil, "", "", 0, "", "warning", "tool-name")
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
	mux.HandleFunc("/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(&github.Repository{
			HTMLURL: github.String("https://test/repo/path"),
		}); err != nil {
			t.Fatal(err)
		}
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(ts.URL + "/")
	g, err := NewGitHubPullRequest(cli, "o", "r", 14, "sha", "warning", "tool-name")
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
