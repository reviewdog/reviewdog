package doghouse

import (
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

// CheckRequest represents doghouse GitHub check request.
type CheckRequest struct {
	// Commit SHA.
	// Required.
	SHA string `json:"sha,omitempty"`
	// PullRequest number.
	// Optional.
	PullRequest int `json:"pull_request,omitempty"`
	// Owner of the repository.
	// Required.
	Owner string `json:"owner,omitempty"`
	// Repository name.
	// Required.
	Repo string `json:"repo,omitempty"`

	// Branch name.
	// Optional.
	// DEPRECATED: No need to fill this field.
	Branch string `json:"branch,omitempty"`

	// Annotations associated with the repository's commit and Pull Request.
	Annotations []*Annotation `json:"annotations,omitempty"`

	// Name of the annotation tool.
	// Optional.
	Name string `json:"name,omitempty"`

	// Level is report level for this request.
	// One of ["info", "warning", "error"]. Default is "error".
	// Optional.
	Level string `json:"level"`

	// Deprecated: Use FilterMode == filter.NoFilter instead.
	//
	// OutsideDiff represents whether it report results in outside diff or not as
	// annotations. It's useful only when PullRequest != 0. If PullRequest is
	// empty, it will always report results all resutls including outside diff
	// (because there are no diff!).
	// Optional.
	OutsideDiff bool `json:"outside_diff"`

	// FilterMode represents a way to filter checks results
	// Optional.
	FilterMode filter.Mode `json:"filter_mode"`
}

// CheckResponse represents doghouse GitHub check response.
type CheckResponse struct {
	// ReportURL is report URL of check run.
	// Optional.
	ReportURL string `json:"report_url,omitempty"`

	// CheckedResults is checked annotations result.
	// This field is expected to be filled for GitHub Actions integration and
	// filled when ReportURL is not available. i.e. reviewdog doesn't have write
	// permission to Check API.
	// It's also not expected to be passed over network via JSON.
	// TODO(haya14busa): Consider to move this type to this package to avoid
	// (cyclic) import.
	// Optional.
	CheckedResults []*filter.FilteredDiagnostic `json:"checked_results"`

	// Conclusion of check result, which is same as GitHub's conclusion of Check
	// API. https://developer.github.com/v3/checks/runs/#parameters-1
	Conclusion string `json:"conclusion,omitempty"`
}

// Annotation represents an annotation to file or specific line.
type Annotation struct {
	// Diagnostic.Location.Path must be relative path to the project root.
	// Optional.
	Diagnostic *rdf.Diagnostic `json:"diagnostic,omitempty"`

	// DEPRECATED fields below. Need to support them for the old reviewdog CLI
	// version.

	// DEPRECATED: Use Diagnostic.
	Path string `json:"path,omitempty"`
	// DEPRECATED: Use Diagnostic.
	Line int `json:"line,omitempty"`
	// DEPRECATED: Use Diagnostic.
	Message string `json:"message,omitempty"`
	// DEPRECATED: Use Diagnostic.
	RawMessage string `json:"raw_message,omitempty"`
}
