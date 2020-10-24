package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
	"github.com/reviewdog/reviewdog/service/bitbucket/openapi"
)

func TestAnnotator(t *testing.T) {
	var testcases = []struct {
		name                string
		runnersList         []string
		comments            []*reviewdog.Comment
		expectedAnnotations map[string]int
		expectedReportCalls map[string][]string
	}{
		{
			name:                "Empty runners list, no comments",
			expectedAnnotations: make(map[string]int),
			expectedReportCalls: make(map[string][]string),
		},
		{
			name:                "Predefined runners list, and no comments",
			runnersList:         []string{"runner1", "runner2"},
			expectedAnnotations: make(map[string]int),
			expectedReportCalls: map[string][]string{
				// since we know list of runners, annotator will create one report for each
				// - on start it will create reports in "pending" state
				// - in the end, it will update report to the "passed" state, since no annotations were found
				reportID("runner1", reporter): {reportResultPending, reportResultPassed},
				reportID("runner2", reporter): {reportResultPending, reportResultPassed},
			},
		},
		{
			name:        "Predefined runners list, and one comment",
			runnersList: []string{"runner1", "runner2"},
			comments: []*reviewdog.Comment{
				newComment("runner2", "main.go", "test", 1),
			},
			expectedAnnotations: map[string]int{
				reportID("runner2", reporter): 1,
			},
			expectedReportCalls: map[string][]string{
				// runner1 has no annotations, so report will be marked as "passed"
				reportID("runner1", reporter): {reportResultPending, reportResultPassed},
				// runner2 has some annotations, so report will be marked as "failed"
				reportID("runner2", reporter): {reportResultPending, reportResultFailed},
			},
		},
		{
			name:        "Predefined runners list, and duplicated comment",
			runnersList: []string{"runner1", "runner2"},
			comments: []*reviewdog.Comment{
				newComment("runner2", "main.go", "test", 1),
				newComment("runner2", "main.go", "test", 1),
			},
			expectedAnnotations: map[string]int{
				reportID("runner2", reporter): 1,
			},
			expectedReportCalls: map[string][]string{
				reportID("runner1", reporter): {reportResultPending, reportResultPassed},
				reportID("runner2", reporter): {reportResultPending, reportResultFailed},
			},
		},
		{
			name:        "Predefined runners list, and many comments",
			runnersList: []string{"runner1", "runner2"},
			comments: func() (comments []*reviewdog.Comment) {
				for i := 0; i < 333; i++ {
					comments = append(comments, newComment("runner1", "main.go", "test", int32(i)))
				}
				return
			}(),
			expectedAnnotations: map[string]int{
				reportID("runner1", reporter): 333,
			},
			expectedReportCalls: map[string][]string{
				reportID("runner1", reporter): {reportResultPending, reportResultFailed},
				reportID("runner2", reporter): {reportResultPending, reportResultPassed},
			},
		},
	}

	username := "test_user"
	repo := "test_repo"
	commit := "test_123"
	urlPrefix := fmt.Sprintf("/repositories/%s/%s/commit/%s/reports/", username, repo, commit)

	for _, test := range testcases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			reportCallsSequence := map[string][]string{}
			annotations := map[string]int{}
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasPrefix(r.RequestURI, urlPrefix) {
					t.Error("Bad request URI", r.RequestURI)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				splt := strings.Split(r.RequestURI, "/")
				if l := len(splt); l < 8 {
					t.Error("Request URI should have 7 parts at least, but got", l)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				reportID := splt[7]
				isPostAnnotationsCall := len(splt) >= 9 && splt[8] == "annotations"

				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				if isPostAnnotationsCall {
					var req []openapi.ReportAnnotation
					if err := json.Unmarshal(body, &req); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusInternalServerError)
					}
					// count how many annotations we created
					annotations[reportID] += len(req)
				} else {
					var req openapi.Report
					if err := json.Unmarshal(body, &req); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusInternalServerError)
					}
					reportCallsSequence[reportID] = append(reportCallsSequence[reportID], *req.Result)
				}

				w.WriteHeader(http.StatusOK)
			}))
			defer ts.Close()

			client := NewAPIClientWithConfigurations(&http.Client{Timeout: 1 * time.Second}, openapi.ServerConfiguration{URL: ts.URL})
			bb := NewReportAnnotator(client, username, repo, commit, test.runnersList)
			ctx := context.Background()

			for _, c := range test.comments {
				if err := bb.Post(ctx, c); err != nil {
					t.Error(err)
				}
			}
			if err := bb.Flush(ctx); err != nil {
				t.Error(err)
			}

			// assert resulted calls
			if !reflect.DeepEqual(test.expectedAnnotations, annotations) {
				t.Errorf("Expected annotations\n%#v\nbut got\n%#v", test.expectedAnnotations, annotations)
			}

			if !reflect.DeepEqual(test.expectedReportCalls, reportCallsSequence) {
				t.Errorf("Expected report calls\n%#v\nbut got\n%#v", test.expectedReportCalls, reportCallsSequence)
			}
		})
	}
}

func newComment(toolName, file, message string, line int32) *reviewdog.Comment {
	return &reviewdog.Comment{
		ToolName: toolName,
		Result: &filter.FilteredDiagnostic{
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: file,
					Range: &rdf.Range{Start: &rdf.Position{
						Line: line,
					}},
				},
				Message: message,
			},
		},
	}
}
