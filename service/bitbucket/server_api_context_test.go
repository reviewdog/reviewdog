package bitbucket

import (
	"context"
	"testing"

	insights "github.com/reva2/bitbucket-insights-api"
)

func TestWithServerVariables(t *testing.T) {
	serverTests := []struct {
		url             string
		protocol        string
		bitbucketDomain string
	}{
		{"http://bitbucket.host.tld", "http", "bitbucket.host.tld"},
		{"https://host.tld", "https", "host.tld"},
		{"http://host.tld/bitbucket", "http", "host.tld/bitbucket"},
		{"https://host.tld/bit/bu/cket", "https", "host.tld/bit/bu/cket"},
		{"http://localhost:7990", "http", "localhost:7990"},
		{"https://localhost:7990/bb", "https", "localhost:7990/bb"},
	}

	for _, server := range serverTests {
		// given
		ctx := context.Background()

		t.Run(server.url, func(t *testing.T) {
			// when
			resCtx, err := withServerVariables(ctx, server.url)

			// then
			if err != nil {
				t.Fatalf("valid url must not cause error")
			}

			serverVariables := resCtx.Value(insights.ContextServerVariables)
			if serverVariables == nil {
				t.Fatalf("serverVariables must not be nil")
			}

			actualProtocol := serverVariables.(map[string]string)["protocol"]
			if actualProtocol != server.protocol {
				t.Fatalf("want %s, but got %s", server.protocol, actualProtocol)
			}

			actualDomain := serverVariables.(map[string]string)["bitbucketDomain"]
			if actualDomain != server.bitbucketDomain {
				t.Fatalf("want %s, but got %s", server.bitbucketDomain, actualDomain)
			}
		})
	}

	wrongServerTests := []string{
		":::",
		"http//bitbucket.my-company.com",
		"http::/bitbucket.my-company.com",
	}

	for _, server := range wrongServerTests {
		// given
		ctx := context.Background()

		t.Run("fail to parse "+server, func(t *testing.T) {
			// when
			resCtx, err := withServerVariables(ctx, server)

			if err == nil {
				t.Fatalf("expect parsing to fail for url: %s, but got %s", server, resCtx.Value(insights.ContextServerVariables))
			}
		})
	}
}
