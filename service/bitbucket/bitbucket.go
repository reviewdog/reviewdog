package bitbucket

import (
	"context"
	"net/http"
	"net/url"
	"time"

	bbapi "github.com/reviewdog/go-bitbucket"
	"golang.org/x/oauth2"
)

const (
	// PipelineProxyURL available while using Bitbucket Pipelines and
	// allows you to use the Reports-API without extra authentication.
	// For that you need to send your request through a proxy server that runs alongside with
	// every pipeline on ‘localhost:29418’, and a valid Auth-Header will automatically be added to your request.
	// https://support.atlassian.com/bitbucket-cloud/docs/code-insights/#Authentication
	// However, if using proxy HTTP API endpoint need to be used
	pipelineProxyURL = "http://localhost:29418"
	pipeProxyURL     = "http://host.docker.internal:29418"
	httpTimeout      = time.Second * 10
)

// NewAPIClient creates Bitbucket API client
func NewAPIClient(isInPipeline bool, isInPipe bool) *bbapi.APIClient {
	httpClient := &http.Client{
		Timeout: httpTimeout,
	}
	server := httpsServer()

	if isInPipeline {
		// if we are on the Bitbucket Pipeline, use HTTP endpoint
		// and proxy
		proxy := pipelineProxyURL
		if isInPipe {
			proxy = pipeProxyURL
		}

		proxyURL, _ := url.Parse(proxy)
		server = httpServer()
		httpClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}
	return NewAPIClientWithConfigurations(httpClient, server)
}

// NewAPIClientWithConfigurations allows to create new Bitbucket API client with
// custom http client or server configurations
func NewAPIClientWithConfigurations(client *http.Client, server bbapi.ServerConfiguration) *bbapi.APIClient {
	config := bbapi.NewConfiguration()
	if client != nil {
		config.HTTPClient = client
	} else {
		config.HTTPClient = &http.Client{
			Timeout: httpTimeout,
		}
	}
	config.Servers = bbapi.ServerConfigurations{server}
	return bbapi.NewAPIClient(config)

}

// WithBasicAuth adds basic auth credentials to context
func WithBasicAuth(ctx context.Context, username, password string) context.Context {
	return context.WithValue(ctx, bbapi.ContextBasicAuth,
		bbapi.BasicAuth{
			UserName: username,
			Password: password,
		})
}

// WithAccessToken adds basic auth credentials to context
func WithAccessToken(ctx context.Context, accessToken string) context.Context {
	return context.WithValue(ctx, bbapi.ContextAccessToken, accessToken)
}

// WithOAuth2 adds basic auth credentials to context
func WithOAuth2(ctx context.Context, tokenSource oauth2.TokenSource) context.Context {
	return context.WithValue(ctx, bbapi.ContextOAuth2, tokenSource)
}

func httpServer() bbapi.ServerConfiguration {
	return bbapi.ServerConfiguration{
		URL: "http://api.bitbucket.org/2.0",
		Description: `If if called from Bitbucket Pipelines,
using HTTP API endpoint and AuthProxy`,
	}
}
func httpsServer() bbapi.ServerConfiguration {
	return bbapi.ServerConfiguration{
		URL:         "https://api.bitbucket.org/2.0",
		Description: `HTTPS API endpoint`,
	}
}
