package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/haya14busa/reviewdog/doghouse"
	"github.com/haya14busa/reviewdog/doghouse/server"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

var githubAppsPrivateKey []byte

const (
	integrationID = 12131 // https://github.com/apps/reviewdog
)

func init() {
	// Private keys https://github.com/settings/apps/reviewdog
	const privateKeyFile = "./secret/github-apps.private-key.pem"
	var err error
	githubAppsPrivateKey, err = ioutil.ReadFile(privateKeyFile)
	if err != nil {
		log.Fatalf("could not read private key: %s", err)
	}
}

func main() {
	http.HandleFunc("/", handleTop)
	http.HandleFunc("/check", handleCheck)
	http.HandleFunc("/webhook", handleWebhook)
	appengine.Main()
}

func handleTop(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "reviewdog")
}

func handleCheck(w http.ResponseWriter, r *http.Request) {
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
		PrivateKey:     githubAppsPrivateKey,
		IntegrationID:  integrationID,
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
