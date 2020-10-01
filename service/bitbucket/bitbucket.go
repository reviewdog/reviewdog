package bitbucket

import (
	"net/http"
	"net/url"
	"time"

	"github.com/reviewdog/reviewdog/service/bitbucket/openapi"
)

const (
	// PipelineProxyURL available while using Bitbucket Pipelines and
	// allows you to use the Reports-API without extra authentication.
	// For that you need to send your request through a proxy server that runs alongside with
	// every pipeline on ‘localhost:29418’, and a valid Auth-Header will automatically be added to your request.
	// https://support.atlassian.com/bitbucket-cloud/docs/code-insights/#Authentication
	// However, if using proxy HTTP API endpoint need to be used
	pipelineProxyURL = "http://localhost:29418"
	httpTimeout      = time.Second * 10
)

var (
	httpServer = openapi.ServerConfiguration{
		URL: "http://api.bitbucket.org/2.0",
		Description: `If if called from Bitbucket Pipelines,
using HTTP API endpoint and AuthProxy`,
	}
	httpsServer = openapi.ServerConfiguration{
		URL:         "https://api.bitbucket.org/2.0",
		Description: `HTTPS API endpoint`,
	}
)

func NewAPIClient(isInPipeline bool) *openapi.APIClient {
	proxyURL, _ := url.Parse(pipelineProxyURL)
	config := openapi.NewConfiguration()
	config.HTTPClient = &http.Client{
		Timeout: httpTimeout,
	}

	if isInPipeline {
		// if we are on the Bitbucket Pipeline, use HTTP endpoint
		// and proxy
		config.Servers = openapi.ServerConfigurations{httpServer}
		config.HTTPClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	} else {
		// TODO: check how to configure credentials
		config.Servers = openapi.ServerConfigurations{httpsServer}
	}

	return openapi.NewAPIClient(config)
}
