package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/haya14busa/reviewdog/doghouse"
	"github.com/haya14busa/reviewdog/doghouse/server"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

type githubChecker struct {
	privateKey    []byte
	integrationID int
}

func (gc *githubChecker) handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := appengine.NewContext(r)

	var req doghouse.CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "failed to decode request: %v", err)
		return
	}

	// Check authorization.
	if !validateCheckRequest(ctx, w, r, req.Owner, req.Repo) {
		return
	}

	opt := &server.NewGitHubClientOption{
		PrivateKey:     gc.privateKey,
		IntegrationID:  gc.integrationID,
		InstallationID: req.InstallationID,
		RepoOwner:      req.Owner,
		RepoName:       req.Repo,
		Client:         urlfetch.Client(ctx),
	}

	gh, err := server.NewGitHubClient(ctx, opt)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err)
		return
	}

	res, err := server.NewChecker(&req, gh).Check(ctx)
	if err != nil {
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

func validateCheckRequest(ctx context.Context, w http.ResponseWriter, r *http.Request, owner, repo string) bool {
	token := extractBearerToken(r)
	if token == "" {
		w.Header().Set("The WWW-Authenticate", `error="invalid_request", error_description="The access token not provided"`)
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	wantToken, err := server.GetRepoToken(ctx, fmt.Sprintf("%s/%s", owner, repo))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return false
	}
	if token != wantToken {
		w.Header().Set("The WWW-Authenticate", `error="invalid_token", error_description="The access token is invalid"`)
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	prefix := "bearer "
	if strings.HasPrefix(strings.ToLower(auth), prefix) {
		return auth[len(prefix):]
	}
	return ""
}
