package main

import (
	"encoding/json"
	"fmt"
	"net/http"

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
	var req doghouse.CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "failed to decode request: %v", err)
		return
	}
	ctx := appengine.NewContext(r)

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
