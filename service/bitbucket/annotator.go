package bitbucket

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/proto/rdf"
	"github.com/reviewdog/reviewdog/service/bitbucket/openapi"
	"github.com/reviewdog/reviewdog/service/commentutil"
)

var _ reviewdog.CommentService = &ReportAnnotator{}

const (
	logoURL  = "https://raw.githubusercontent.com/haya14busa/i/d598ed7dc49fefb0018e422e4c43e5ab8f207a6b/reviewdog/reviewdog.logo.png"
	reporter = "reviewdog"

	annotationTypeCodeSmell     = "CODE_SMELL"
	annotationTypeVulnerability = "VULNERABILITY"
	annotationTypeBug           = "BUG"

	annotationResultPassed  = "PASSED"
	annotationResultFailed  = "FAILED"
	annotationResultSkipped = "SKIPPED"
	annotationResultIgnored = "IGNORED"
	annotationResultPending = "PENDING"

	annotationSeverityHigh     = "HIGH"
	annotationSeverityMedium   = "MEDIUM"
	annotationSeverityLow      = "LOW"
	annotationSeverityCritical = "CRITICAL"

	reportTypeSecurity = "SECURITY"
	reportTypeCoverage = "COVERAGE"
	reportTypeTest     = "TEST"
	reportTypeBug      = "BUG"

	reportDataTypeBool       = "BOOLEAN"
	reportDataTypeDate       = "DATE"
	reportDataTypeDuration   = "DURATION"
	reportDataTypeLink       = "LINK"
	reportDataTypeNumber     = "NUMBER"
	reportDataTypePercentage = "PERCENTAGE"
	reportDataTypeText       = "TEXT"

	reportResultPassed  = "PASSED"
	reportResultFailed  = "FAILED"
	reportResultPending = "PENDING"
)

var severityMap = map[rdf.Severity]string{
	rdf.Severity_INFO:    annotationSeverityLow,
	rdf.Severity_WARNING: annotationSeverityMedium,
	rdf.Severity_ERROR:   annotationSeverityHigh,
}

// ReportAnnotator is a comment service for Bitbucket Code Insights reports.
//
// API:
//  https://developer.atlassian.com/bitbucket/api/2/reference/resource/repositories/%7Bworkspace%7D/%7Brepo_slug%7D/commit/%7Bcommit%7D/reports/%7BreportId%7D/annotations#post
//  POST /2.0/repositories/{workspace}/{repo_slug}/commit/{commit}/reports/{reportId}/annotations
type ReportAnnotator struct {
	cli         *openapi.APIClient
	sha         string
	owner, repo string
	reportTitle string

	muComments   sync.Mutex
	postComments []*reviewdog.Comment

	// postedcs commentutil.PostedComments

	// wd is working directory relative to root of repository.
	wd       string
	reportID string
}

// NewReportAnnotator creates new Bitbucket Report Annotator
func NewReportAnnotator(cli *openapi.APIClient, reportTitle, owner, repo, sha string) *ReportAnnotator {
	if reportTitle == "" {
		reportTitle = "Reviewdog Report"
	}

	return &ReportAnnotator{
		cli:         cli,
		reportTitle: reportTitle,
		sha:         sha,
		owner:       owner,
		repo:        repo,
		reportID:    reporter + "-" + reportTitle,
	}
}

// Post accepts a comment and holds it. Flush method actually posts comments to
// Bitbucket in batch.
func (r *ReportAnnotator) Post(_ context.Context, c *reviewdog.Comment) error {
	c.Result.Diagnostic.GetLocation().Path = filepath.ToSlash(
		filepath.Join(r.wd, c.Result.Diagnostic.GetLocation().GetPath()))
	r.muComments.Lock()
	defer r.muComments.Unlock()
	r.postComments = append(r.postComments, c)
	return nil
}

// Flush posts comments which has not been posted yet.
func (r *ReportAnnotator) Flush(ctx context.Context) error {
	r.muComments.Lock()
	defer r.muComments.Unlock()

	var annotations []openapi.ReportAnnotation

	issuesCount := map[rdf.Severity]int{}

	for _, c := range r.postComments {
		issuesCount[c.Result.Diagnostic.GetSeverity()]++
		annotations = append(annotations, annotationFromReviewDogComment(*c))
	}

	if len(annotations) == 0 {
		return r.createOrUpdateReport(ctx, reportResultPassed)
	}

	reportStatus := reportResultPending
	if issuesCount[rdf.Severity_ERROR] > 0 {
		reportStatus = reportResultFailed
	}

	if err := r.createOrUpdateReport(ctx, reportStatus); err != nil {
		return err
	}

	_, resp, err := r.cli.ReportsApi.BulkCreateOrUpdateAnnotations(
		ctx, r.owner, r.repo, r.sha, r.reportID,
	).Body(annotations).Execute()

	if err != nil {
		return err
	}

	return checkHTTPResp(resp, http.StatusOK)
}

func annotationFromReviewDogComment(c reviewdog.Comment) openapi.ReportAnnotation {
	a := openapi.NewReportAnnotation()
	switch c.ToolName {
	// TODO: different type of annotation based on tool?
	default:
		a.SetAnnotationType(annotationTypeCodeSmell)
	}

	a.SetSummary(c.Result.Diagnostic.GetMessage())
	a.SetDetails(commentutil.MarkdownComment(&c))
	a.SetLine(c.Result.Diagnostic.GetLocation().GetRange().GetStart().GetLine())
	a.SetPath(c.Result.Diagnostic.GetLocation().GetPath())
	if v, ok := severityMap[c.Result.Diagnostic.GetSeverity()]; ok {
		a.SetSeverity(v)
	}
	a.SetLink(c.Result.Diagnostic.GetCode().GetUrl())

	return *a
}

func (r *ReportAnnotator) createOrUpdateReport(ctx context.Context, status string) error {
	var report = openapi.NewReport()
	report.SetTitle(r.reportTitle)
	// TODO: different report types?
	report.SetReportType(reportTypeBug)
	report.SetReporter(reporter)
	report.SetLogoUrl(logoURL)
	report.SetResult(status)

	_, resp, err := r.cli.ReportsApi.CreateOrUpdateReport(
		ctx, r.owner, r.repo, r.sha, r.reportID,
	).Body(*report).Execute()

	if err != nil {
		return err
	}

	return checkHTTPResp(resp, http.StatusOK)
}

func checkHTTPResp(resp *http.Response, expectedCode int) error {
	if resp.StatusCode != expectedCode {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		msg := fmt.Sprintf("Received unexpected %d code from Bitbucket API", resp.StatusCode)
		if len(body) > 0 {
			msg += " with message:\n" + string(body)
		}
		return errors.New(msg)
	}

	return nil
}
