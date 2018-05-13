package doghouse

// GitHub Check Request
type CheckRequest struct {
	// Installation ID of reviewdog GitHub APP.
	// https://github.com/settings/installations/<INSTALLATION_ID>
	// You can find from here: https://github.com/settings/installations/
	InstallationID int `json:"installation_id,omitempty"`

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
