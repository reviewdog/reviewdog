package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/vvakame/sdlog/aelog"

	"github.com/reviewdog/reviewdog/doghouse"
	"github.com/reviewdog/reviewdog/doghouse/server"
	"github.com/reviewdog/reviewdog/doghouse/server/ciutil"
	"github.com/reviewdog/reviewdog/doghouse/server/storage"
)

type githubChecker struct {
	privateKey       []byte
	integrationID    int
	ghInstStore      storage.GitHubInstallationStore
	ghRepoTokenStore storage.GitHubRepositoryTokenStore
	tr               http.RoundTripper
}

func (gc *githubChecker) handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	var req doghouse.CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "failed to decode request: %v", err)
		return
	}

	// Check authorization.
	if !gc.validateCheckRequest(ctx, w, r, req.Owner, req.Repo) {
		return
	}

	opt := &server.NewGitHubClientOption{
		PrivateKey:    gc.privateKey,
		IntegrationID: gc.integrationID,
		RepoOwner:     req.Owner,
		Client: &http.Client{
			Transport: gc.tr,
		},
	}

	gh, err := server.NewGitHubClient(ctx, opt)
	if err != nil {
		aelog.Errorf(ctx, "failed to create GitHub client: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err)
		return
	}

	res, err := server.NewChecker(&req, gh).Check(ctx)
	if err != nil {
		aelog.Errorf(ctx, "failed to run checker: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err)
		return
	}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err)
		return
	}
}

func (gc *githubChecker) validateCheckRequest(ctx context.Context, w http.ResponseWriter, r *http.Request, owner, repo string) bool {
	if extractBearerToken(r) == "" {
		// Update Travis IP Address before checking IP to reduce the # of
		// flaky errors when token is not present.
		if err := ciutil.UpdateTravisCIIPAddrs(&http.Client{}); err != nil {
			aelog.Errorf(ctx, "failed to update travis CI IP addresses: %v", err)
		}
	}
	aelog.Infof(ctx, "Remote Addr: %s", ciutil.IPFromReq(r))
	if ciutil.IsFromCI(r) {
		// Skip token validation if it's from trusted CI providers.
		return true
	}
	return gc.validateCheckToken(ctx, w, r, owner, repo)
}

func (gc *githubChecker) validateCheckToken(ctx context.Context, w http.ResponseWriter, r *http.Request, owner, repo string) bool {
	token := extractBearerToken(r)
	if token == "" {
		w.Header().Set("The WWW-Authenticate", `error="invalid_request", error_description="The access token not provided"`)
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "The access token not provided. Get token from %s", githubRepoURL(ctx, r, owner, repo))
		return false
	}
	_, wantToken, err := gc.ghRepoTokenStore.Get(ctx, owner, repo)
	if err != nil {
		aelog.Errorf(ctx, "failed to get repository (%s/%s) token: %v", owner, repo, err)
	}
	if wantToken == nil {
		w.WriteHeader(http.StatusNotFound)
		return false
	}
	if token != wantToken.Token {
		w.Header().Set("The WWW-Authenticate", `error="invalid_token", error_description="The access token is invalid"`)
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "The access token is invalid. Get valid token from %s", githubRepoURL(ctx, r, owner, repo))
		return false
	}
	return true
}

func githubRepoURL(ctx context.Context, r *http.Request, owner, repo string) string {
	u := doghouseBaseURL(ctx, r)
	u.Path = fmt.Sprintf("/gh/%s/%s", owner, repo)
	return u.String()
}

func doghouseBaseURL(ctx context.Context, r *http.Request) *url.URL {
	scheme := ""
	if r.URL != nil && r.URL.Scheme != "" {
		scheme = r.URL.Scheme
	}
	if scheme == "" {
		scheme = "https"
	}
	u, err := url.Parse(scheme + "://" + r.Host)
	if err != nil {
		aelog.Errorf(ctx, "%v", err)
	}
	return u
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	prefix := "bearer "
	if strings.HasPrefix(strings.ToLower(auth), prefix) {
		return auth[len(prefix):]
	}
	return ""
}
