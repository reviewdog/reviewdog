package bitbucket

import (
	"fmt"
	insights "github.com/reva2/bitbucket-insights-api"
	"github.com/reviewdog/reviewdog"
)

// ServerAPIHelper is collection of utility functions used to build requests
// for Bitbucket Server Code Insights API
type ServerAPIHelper struct{}

// BuildReport builds Code Insights API report object
func (h *ServerAPIHelper) BuildReport(req *ReportRequest) insights.Report {
	data := insights.NewReport(req.Title)

	data.SetReporter(req.Reporter)
	data.SetLogoUrl(req.LogoURL)
	data.SetResult(h.convertResult(req.Result))
	data.SetDetails(req.Details)

	return *data
}

// BuildAnnotations builds list of Code Insights API annotation objects for specified comments
func (h *ServerAPIHelper) BuildAnnotations(comments []*reviewdog.Comment) insights.AnnotationsList {
	annotations := make([]insights.Annotation, len(comments))
	for idx, comment := range comments {
		annotations[idx] = h.buildAnnotation(comment)
	}

	list := insights.NewAnnotationsList(annotations)

	return *list
}

func (h *ServerAPIHelper) buildAnnotation(comment *reviewdog.Comment) insights.Annotation {
	severity := convertSeverity(comment.Result.Diagnostic.GetSeverity())
	if severity == "" {
		severity = annotationSeverityLow
	}

	data := insights.NewAnnotation(
		comment.Result.Diagnostic.GetLocation().GetPath(),
		comment.Result.Diagnostic.GetLocation().GetRange().GetStart().GetLine()-1,
		fmt.Sprintf(`[%s] %s`, comment.ToolName, comment.Result.Diagnostic.GetMessage()),
		severity,
	)
	data.SetExternalId(externalIDFromDiagnostic(comment.Result.Diagnostic))
	data.SetType(annotationTypeCodeSmell)

	if link := comment.Result.Diagnostic.GetCode().GetUrl(); link != "" {
		data.SetLink(link)
	}

	return *data
}

func (h *ServerAPIHelper) convertResult(result string) string {
	switch result {
	case reportResultFailed:
		return "FAIL"

	default:
		return "PASS"
	}
}
