package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/go-github/github"
	"github.com/haya14busa/reviewdog/doghouse/server"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

type githubWebhookHandler struct {
	secret []byte
}

func (g *githubWebhookHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := appengine.NewContext(r)
	payload, err := github.ValidatePayload(r, g.secret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	switch github.WebHookType(r) {
	case "check_suite":
		if err := handleCheckSuiteEvent(ctx, payload); err != nil {
			log.Errorf(ctx, "failed to handle check_suite event: %v", err)
		}
	}
}

func handleCheckSuiteEvent(ctx context.Context, payload []byte) error {
	var c server.CheckSuiteEvent
	if err := json.Unmarshal(payload, &c); err != nil {
		return err
	}
	log.Debugf(ctx, "%#v", c)
	return server.SaveInstallationFromCheckSuite(ctx, c)
}
