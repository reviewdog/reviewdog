package bitbucket

import (
	"fmt"

	bbapi "github.com/reviewdog/go-bitbucket"
	"github.com/sezzle/reviewdog"
)

// CloudAPIHelper is collection of utility functions used to build requests
// for Bitbucket Cloud Code Insights API
type CloudAPIHelper struct{}

// BuildReport builds Code Insights API report object
func (c *CloudAPIHelper) BuildReport(req *ReportRequest) bbapi.Report {
	data := bbapi.NewReport()
	data.SetTitle(req.Title)
	data.SetReportType(req.Type)
	data.SetReporter(req.Reporter)
	data.SetLogoUrl(req.LogoURL)
	data.SetResult(req.Result)
	data.SetDetails(req.Details)

	return *data
}

// BuildAnnotations builds list of Code Insights API annotation objects for specified comments
func (c *CloudAPIHelper) BuildAnnotations(comments []*reviewdog.Comment) []bbapi.ReportAnnotation {
	annotations := make([]bbapi.ReportAnnotation, len(comments))
	for idx, comment := range comments {
		annotations[idx] = c.buildAnnotation(comment)
	}

	return annotations
}

func (c *CloudAPIHelper) buildAnnotation(comment *reviewdog.Comment) bbapi.ReportAnnotation {
	data := bbapi.NewReportAnnotation()
	data.SetExternalId(externalIDFromDiagnostic(comment.Result.Diagnostic))
	data.SetAnnotationType(annotationTypeCodeSmell)
	data.SetSummary(comment.Result.Diagnostic.GetMessage())
	data.SetDetails(fmt.Sprintf(`[%s] %s`, comment.ToolName, comment.Result.Diagnostic.GetMessage()))
	data.SetLine(comment.Result.Diagnostic.GetLocation().GetRange().GetStart().GetLine())
	data.SetPath(comment.Result.Diagnostic.GetLocation().GetPath())

	if severity := convertSeverity(comment.Result.Diagnostic.GetSeverity()); severity != "" {
		data.SetSeverity(severity)
	}

	if link := comment.Result.Diagnostic.GetCode().GetUrl(); link != "" {
		data.SetLink(link)
	}

	return *data
}
