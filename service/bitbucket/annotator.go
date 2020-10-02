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
//  POST /2.0/repositories/{username}/{repo_slug}/commit/{commit}/reports/{reportId}/annotations
type ReportAnnotator struct {
	ctx         context.Context
	cli         *openapi.APIClient
	sha         string
	owner, repo string
	reportTitle string

	muAnnotations sync.Mutex
	annotations   []openapi.ReportAnnotation
	issuesCount   map[rdf.Severity]int

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
		reportID:    reporter + "-" + strings.ReplaceAll(reportTitle, " ", "_"),
	}
}

// Post accepts a comment and holds it. Flush method actually posts comments to
// Bitbucket in batch.
func (r *ReportAnnotator) Post(_ context.Context, c *reviewdog.Comment) error {
	c.Result.Diagnostic.GetLocation().Path = filepath.ToSlash(
		filepath.Join(r.wd, c.Result.Diagnostic.GetLocation().GetPath()))
	r.muAnnotations.Lock()
	defer r.muAnnotations.Unlock()

	r.issuesCount[c.Result.Diagnostic.GetSeverity()]++
	r.annotations = append(r.annotations, annotationFromReviewDogComment(*c))

	return nil
}

// Flush posts comments which has not been posted yet.
func (r *ReportAnnotator) Flush(ctx context.Context) error {
	r.muAnnotations.Lock()
	defer r.muAnnotations.Unlock()

	if len(r.annotations) == 0 {
		return r.createOrUpdateReport(ctx, reportResultPassed)
	}

	reportStatus := reportResultPending
	if r.issuesCount[rdf.Severity_ERROR] > 0 {
		reportStatus = reportResultFailed
	}

	if err := r.createOrUpdateReport(ctx, reportStatus); err != nil {
		return err
	}

	_, resp, err := r.cli.ReportsApi.BulkCreateOrUpdateAnnotations(
		ctx, r.owner, r.repo, r.sha, r.reportID,
	).Body(r.annotations).Execute()

	if err := checkAPIError(err, resp, http.StatusOK); err != nil {
		return fmt.Errorf("bitbucket.BulkCreateOrUpdateAnnotations: %s", err)
	}

	return nil
}

func annotationFromReviewDogComment(c reviewdog.Comment) openapi.ReportAnnotation {
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
	if v, ok := severityMap[c.Result.Diagnostic.GetSeverity()]; ok {
		a.SetSeverity(v)
	}
	if link := c.Result.Diagnostic.GetCode().GetUrl(); link != "" {
		a.SetLink(link)
	}

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
	report.SetDetails("Woof-Woof! This report generated for you by reviewdog")

	_, resp, err := r.cli.ReportsApi.CreateOrUpdateReport(
		ctx, r.owner, r.repo, r.sha, r.reportID,
	).Body(*report).Execute()

	if err := checkAPIError(err, resp, http.StatusOK); err != nil {
		return fmt.Errorf("bitbucket.CreateOrUpdateReport: %s", err)
	}

	return nil
}

func hashString(str string) string {
	h := sha256.New()
	h.Write([]byte(str))
	return fmt.Sprintf("%x", h.Sum(nil))
}
