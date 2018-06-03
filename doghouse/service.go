package doghouse

// CheckRequest represents doghouse GitHub check request.
type CheckRequest struct {
	// Commit SHA.
	// Required.
	SHA string `json:"sha,omitempty"`
	// PullRequest number.
	// Required.
	PullRequest int `json:"pull_request,omitempty"`
	// Owner of the repository.
	// Required.
	Owner string `json:"owner,omitempty"`
	// Repository name.
	// Required.
	Repo string `json:"repo,omitempty"`
	// Branch name.
	// Optional.
	Branch string `json:"branch,omitempty"`

	// Annotations associated with the repository's commit and Pull Request.
	Annotations []*Annotation `json:"annotations,omitempty"`

	// Name of the annotation tool.
	// Optional.
	Name string `json:"name,omitempty"`
}

// CheckResponse represents doghouse GitHub check response.
type CheckResponse struct {
	ReportURL string `json:"report_url,omitempty"`
}

// Annotation represents an annotaion to file or specific line.
type Annotation struct {
	// Relative file path
	// Required.
	Path string `json:"path,omitempty"`
	// Line number.
	// Optional.
	Line int `json:"line,omitempty"`
	// Annotation message.
	// Required.
	Message string `json:"message,omitempty"`
	// Original error message of this annotaion.
	// Optional.
	RawMessage string `json:"raw_message,omitempty"`
}
