package doghouse

// GitHub Check Request
type CheckRequest struct {
	SHA         string `json:"sha,omitempty"`
	PullRequest int    `json:"pull_request,omitempty"`
	Owner       string `json:"owner,omitempty"`
	Repo        string `json:"repo,omitempty"`

	// Name of the annotation tool.
	// Optional.
	Name        string        `json:"name,omitempty"`
	Annotations []*Annotation `json:"annotations,omitempty"`
}

// GitHub Check Response
type CheckResponse struct {
	ReportURL string `json:"report_url,omitempty"`
}

type Annotation struct {
	Path       string `json:"path,omitempty"`        // relative file path
	Line       int    `json:"line,omitempty"`        // line number
	Message    string `json:"message,omitempty"`     // message
	RawMessage string `json:"raw_message,omitempty"` // original error message
}
