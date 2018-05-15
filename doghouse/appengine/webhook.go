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

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := appengine.NewContext(r)
	switch github.WebHookType(r) {
	case "check_suite":
		if err := handleCheckSuiteEvent(ctx, r); err != nil {
			log.Errorf(ctx, "failed to handle check_suite event: %v", err)
		}
	}
}

func handleCheckSuiteEvent(ctx context.Context, r *http.Request) error {
	var c server.CheckSuiteEvent
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		return err
	}
	log.Debugf(ctx, "%#v", c)
	return server.SaveInstallationFromCheckSuite(ctx, c)
}
