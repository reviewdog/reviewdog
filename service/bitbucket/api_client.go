package bitbucket

import (
	"context"
	"fmt"

	"github.com/reviewdog/reviewdog"
)

// ReportRequest is an object that represent parameters used to create/update report
type ReportRequest struct {
	Owner      string
	Repository string
	Commit     string
	ReportID   string
	Type       string
	Title      string
	Reporter   string
	Result     string
	Details    string
	LogoURL    string
}

// AnnotationsRequest is an object that represent parameters used to create/update annotations
type AnnotationsRequest struct {
	Owner      string
	Repository string
	Commit     string
	ReportID   string
	Comments   []*reviewdog.Comment
}

// APIClient is client for Bitbucket Code Insights API
type APIClient interface {

	// CreateOrUpdateReport creates or updates specified report
	CreateOrUpdateReport(ctx context.Context, req *ReportRequest) error

	// CreateOrUpdateAnnotations creates or updates annotations
	CreateOrUpdateAnnotations(ctx context.Context, req *AnnotationsRequest) error
}

// UnexpectedResponseError is triggered when we have unexpected response from Code Insights API
type UnexpectedResponseError struct {
	Code int
	Body []byte
}

func (e UnexpectedResponseError) Error() string {
	msg := fmt.Sprintf("received unexpected %d code from Bitbucket API", e.Code)

	if len(e.Body) > 0 {
		msg += " with message:\n" + string(e.Body)
	}

	return msg
}
