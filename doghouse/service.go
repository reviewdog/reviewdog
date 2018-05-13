package doghouse

// GitHub Check Request
type CheckRequest struct {
	// Installation ID of reviewdog GitHub APP.
	// https://github.com/settings/installations/<INSTALLATION_ID>
	// You can find from here: https://github.com/settings/installations/
	InstallationID int `json:"installation_id"`

	SHA         string `json:"sha"`
	PullRequest int    `json:"pull_request"`
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`

	// Name of the annotation tool.
	// Optional.
	Name        string        `json:"name"`
	Annotations []*Annotation `json:"annotations"`
}

// GitHub Check Response
type CheckResponse struct {
	ReportURL string `json:"report_url"`
}

type Annotation struct {
	Path       string `json:"path"`        // relative file path
	Line       int    `json:"line"`        // line number
	Message    string `json:"message"`     // message
	RawMessage string `json:"raw_message"` // original error message
}
