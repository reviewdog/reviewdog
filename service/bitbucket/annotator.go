package bitbucket

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/proto/rdf"
	"github.com/reviewdog/reviewdog/service/bitbucket/openapi"
)

var _ reviewdog.CommentService = &ReportAnnotator{}

const (
	// avatar from https://github.com/apps/reviewdog
	logoURL  = "https://avatars1.githubusercontent.com/in/12131"
	reporter = "reviewdog"
)

// ReportAnnotator is a comment service for Bitbucket Code Insights reports.
//
// API:
//  https://developer.atlassian.com/bitbucket/api/2/reference/resource/repositories/%7Bworkspace%7D/%7Brepo_slug%7D/commit/%7Bcommit%7D/reports/%7BreportId%7D/annotations#post
//  POST /2.0/repositories/{username}/{repo_slug}/commit/{commit}/reports/{reportId}/annotations
type ReportAnnotator struct {
	cli         *openapi.APIClient
	sha         string
	owner, repo string

	muAnnotations sync.Mutex
	annotations   map[string][]openapi.ReportAnnotation
	severityMap   map[rdf.Severity]string

	// wd is working directory relative to root of repository.
	wd string
}

// NewReportAnnotator creates new Bitbucket Report Annotator
func NewReportAnnotator(cli *openapi.APIClient, owner, repo, sha string) *ReportAnnotator {
	return &ReportAnnotator{
		cli:         cli,
		sha:         sha,
		owner:       owner,
		repo:        repo,
		annotations: make(map[string][]openapi.ReportAnnotation),
		severityMap: map[rdf.Severity]string{
			rdf.Severity_INFO:    annotationSeverityLow,
			rdf.Severity_WARNING: annotationSeverityMedium,
			rdf.Severity_ERROR:   annotationSeverityHigh,
		},
	}
}

// Post accepts a comment and holds it. Flush method actually posts comments to
// Bitbucket in batch.
func (r *ReportAnnotator) Post(_ context.Context, c *reviewdog.Comment) error {
	c.Result.Diagnostic.GetLocation().Path = filepath.ToSlash(
		filepath.Join(r.wd, c.Result.Diagnostic.GetLocation().GetPath()))
	r.muAnnotations.Lock()
	defer r.muAnnotations.Unlock()

	r.annotations[c.ToolName] = append(r.annotations[c.ToolName], r.annotationFromReviewDogComment(*c))

	return nil
}

// Flush posts comments which has not been posted yet.
func (r *ReportAnnotator) Flush(ctx context.Context) error {
	r.muAnnotations.Lock()
	defer r.muAnnotations.Unlock()

	// create/update/annotate report per tool
	for tool, annotations := range r.annotations {
		reportID := reportID(reporter, tool)
		title := fmt.Sprintf("[%s] %s report", tool, reporter)
		if len(r.annotations) == 0 {
			// if no annotation, create Passed report
			if err := r.createOrUpdateReport(ctx, reportID, title, reportResultPassed); err != nil {
				return err
			}
			// and move one
			continue
		}

		// create report or update report first, with the failed status
		if err := r.createOrUpdateReport(ctx, reportID, title, reportResultFailed); err != nil {
			return err
		}

		// add annotations to the report
		_, resp, err := r.cli.ReportsApi.BulkCreateOrUpdateAnnotations(
			ctx, r.owner, r.repo, r.sha, reportID,
		).Body(annotations).Execute()

		if err := checkAPIError(err, resp, http.StatusOK); err != nil {
			return fmt.Errorf("bitbucket.BulkCreateOrUpdateAnnotations: %s", err)
		}
	}

	return nil
}

func (r *ReportAnnotator) annotationFromReviewDogComment(c reviewdog.Comment) openapi.ReportAnnotation {
	a := openapi.NewReportAnnotation()
	switch c.ToolName {
	// TODO: different type of annotation based on tool?
	default:
		a.SetAnnotationType(annotationTypeCodeSmell)
	}

	// hash the output of linter and use it as external id
	a.SetExternalId(hashString(c.Result.Diagnostic.OriginalOutput))
	a.SetSummary(c.Result.Diagnostic.GetMessage())
	a.SetDetails(fmt.Sprintf(`[%s] %s`, c.ToolName, c.Result.Diagnostic.GetMessage()))
	a.SetLine(c.Result.Diagnostic.GetLocation().GetRange().GetStart().GetLine())
	a.SetPath(c.Result.Diagnostic.GetLocation().GetPath())
	if v, ok := r.severityMap[c.Result.Diagnostic.GetSeverity()]; ok {
		a.SetSeverity(v)
	}
	if link := c.Result.Diagnostic.GetCode().GetUrl(); link != "" {
		a.SetLink(link)
	}

	return *a
}

func (r *ReportAnnotator) createOrUpdateReport(ctx context.Context, redportID, title, reportStatus string) error {
	var report = openapi.NewReport()
	report.SetTitle(title)
	// TODO: different report types?
	report.SetReportType(reportTypeBug)
	report.SetReporter(reporter)
	report.SetLogoUrl(logoURL)
	report.SetResult(reportStatus)
	report.SetDetails("Woof-Woof! This report generated for you by reviewdog")

	_, resp, err := r.cli.ReportsApi.CreateOrUpdateReport(
		ctx, r.owner, r.repo, r.sha, redportID,
	).Body(*report).Execute()

	if err := checkAPIError(err, resp, http.StatusOK); err != nil {
		return fmt.Errorf("bitbucket.CreateOrUpdateReport: %s", err)
	}

	return nil
}

func hashString(str string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(str))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func reportID(ids ...string) string {
	return strings.ReplaceAll(strings.ToLower(strings.Join(ids, "-")), " ", "_")
}
