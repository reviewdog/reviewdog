package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/google/go-github/v26/github"
	"github.com/reviewdog/reviewdog/doghouse/server/storage"
	"google.golang.org/appengine"
)

type githubWebhookHandler struct {
	secret      []byte
	ghInstStore storage.GitHubInstallationStore
}

func (g *githubWebhookHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := appengine.NewContext(r)
	payload, err := g.validatePayload(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var handleFunc func(ctx context.Context, payload []byte) (status int, err error)
	switch github.WebHookType(r) {
	case "installation":
		handleFunc = g.handleInstallationEvent
	case "check_suite":
		handleFunc = g.handleCheckSuiteEvent
	}
	if handleFunc != nil {
		status, err := handleFunc(ctx, payload)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "failed to handle %s event: %v", github.WebHookType(r), err)
			return
		}
		w.WriteHeader(status)
		fmt.Fprintf(w, "resource created. event: %s", github.WebHookType(r))
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (g *githubWebhookHandler) validatePayload(r *http.Request) (payload []byte, err error) {
	if appengine.IsDevAppServer() {
		return ioutil.ReadAll(r.Body)
	}
	return github.ValidatePayload(r, g.secret)
}

func (g *githubWebhookHandler) handleCheckSuiteEvent(ctx context.Context, payload []byte) (status int, err error) {
	var c CheckSuiteEvent
	if err := json.Unmarshal(payload, &c); err != nil {
		return 0, err
	}
	switch c.Action {
	case "requested":
		// Update InstallationID on check_suite event in case the users re-install
		// the app.
		return http.StatusCreated, g.ghInstStore.Put(ctx, &storage.GitHubInstallation{
			InstallationID: c.Installation.ID,
			AccountName:    c.Repository.Owner.Login,
			AccountID:      c.Repository.Owner.ID,
		})
	}
	return http.StatusAccepted, nil
}

func (g *githubWebhookHandler) handleInstallationEvent(ctx context.Context, payload []byte) (status int, err error) {
	var e InstallationEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return 0, err
	}
	switch e.Action {
	case "created":
		return http.StatusCreated, g.ghInstStore.Put(ctx, &storage.GitHubInstallation{
			InstallationID: e.Installation.ID,
			AccountName:    e.Installation.Account.Login,
			AccountID:      e.Installation.Account.ID,
		})
	}
	return http.StatusAccepted, nil
}

// Example: https://gist.github.com/haya14busa/7a9a87da5159d6853fed865ca5ad5ec7
type InstallationEvent struct {
	Action       string `json:"action,omitempty"`
	Installation struct {
		ID      int64 `json:"id,omitempty"`
		Account struct {
			Login string `json:"login"`
			ID    int64  `json:"id"`
		} `json:"account"`
	} `json:"installation,omitempty"`
}

// Example: https://gist.github.com/haya14busa/2aaffaa89a224ee2ffcbd3d414d6d009
type CheckSuiteEvent struct {
	Action     string `json:"action,omitempty"`
	Repository struct {
		ID       int64  `json:"id,omitempty"`
		FullName string `json:"full_name,omitempty"`
		Owner    struct {
			Login string `json:"login"`
			ID    int64  `json:"id"`
		} `json:"owner"`
	} `json:"repository,omitempty"`
	Installation struct {
		ID int64 `json:"id,omitempty"`
	} `json:"installation,omitempty"`
}
