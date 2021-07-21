package bitbucket

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	insights "github.com/reva2/bitbucket-insights-api"
)

// ServerAPIClient is wrapper for Bitbucket Server Code Insights API client
type ServerAPIClient struct {
	cli    *insights.APIClient
	helper *ServerAPIHelper
}

// NewServerAPIClient creates client for Bitbucket Server Code Insights API
func NewServerAPIClient() APIClient {
	httpClient := &http.Client{
		Timeout: httpTimeout,
	}

	config := insights.NewConfiguration()
	config.HTTPClient = httpClient

	return &ServerAPIClient{
		cli:    insights.NewAPIClient(config),
		helper: &ServerAPIHelper{},
	}
}

// CreateOrUpdateReport creates or updates specified report
func (c *ServerAPIClient) CreateOrUpdateReport(ctx context.Context, req *ReportRequest) error {
	// Bitbucket Server API doesn't support pending status
	if req.Result == reportResultPending {
		return nil
	}

	// We need to drop report and delete all annotations created previously
	err := c.deleteReport(ctx, req)
	if err != nil {
		return err
	}

	_, resp, err := c.cli.InsightsApi.
		UpdateReport(ctx, req.Owner, req.Repository, req.Commit, req.ReportID).
		Report(c.helper.BuildReport(req)).
		Execute()

	if err := c.checkAPIError(err, resp, http.StatusOK); err != nil {
		return fmt.Errorf("failed to create code insights report: %w", err)
	}

	return nil
}

// CreateOrUpdateAnnotations creates or updates annotations
func (c *ServerAPIClient) CreateOrUpdateAnnotations(ctx context.Context, req *AnnotationsRequest) error {
	resp, err := c.cli.InsightsApi.
		CreateAnnotations(ctx, req.Owner, req.Repository, req.Commit, req.ReportID).
		AnnotationsList(c.helper.BuildAnnotations(req.Comments)).
		Execute()

	if err := c.checkAPIError(err, resp, http.StatusNoContent); err != nil {
		return fmt.Errorf("failed to create annotations: %w", err)
	}

	return nil
}

func (c *ServerAPIClient) deleteReport(ctx context.Context, report *ReportRequest) error {
	resp, err := c.cli.InsightsApi.
		DeleteReport(ctx, report.Owner, report.Repository, report.Commit, report.ReportID).
		Execute()

	if err := c.checkAPIError(err, resp, http.StatusNoContent); err != nil {
		return fmt.Errorf("failted to delete code insights report: %w", err)
	}

	return nil
}

func (c *ServerAPIClient) checkAPIError(err error, resp *http.Response, expectedCode int) error {
	if err != nil {
		return fmt.Errorf("bitubucket API error: %w", err)
	}

	if resp != nil && resp.StatusCode != expectedCode {
		body, _ := ioutil.ReadAll(resp.Body)

		return UnexpectedResponseError{
			Code: resp.StatusCode,
			Body: body,
		}
	}

	return nil
}
