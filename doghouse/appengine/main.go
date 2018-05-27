package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/haya14busa/reviewdog/doghouse"
	"github.com/haya14busa/reviewdog/doghouse/server"
	"github.com/haya14busa/reviewdog/doghouse/server/cookieman"
	"github.com/haya14busa/secretbox"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

var (
	githubAppsPrivateKey []byte
	githubWebhookSecret  []byte
	cookiemanager        *cookieman.CookieMan
)

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
	s := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if s == "" {
		log.Fatalf("GITHUB_WEBHOOK_SECRET is not set")
	}
	githubWebhookSecret = []byte(s)
	initCookieMan()
}

func initCookieMan() {
	// Create secret key by following command.
	// $ ruby -rsecurerandom -e 'puts SecureRandom.hex(32)'
	cipher, err := secretbox.NewFromHexKey(os.Getenv("SECRETBOX_SECRET"))
	if err != nil {
		log.Fatalf("failed to create secretbox: %v", err)
	}
	c := cookieman.CookieOption{
		http.Cookie{
			HttpOnly: true,
			Secure:   !appengine.IsDevAppServer(),
			MaxAge:   int((30 * 24 * time.Hour).Seconds()),
			Path:     "/",
		},
	}
	if !appengine.IsDevAppServer() {
		c.Secure = true
		c.Domain = "review-dog.appspot.com"
	}
	cookiemanager = cookieman.New(cipher, c)
}

func main() {
	ghHandler := NewGitHubHandler(
		os.Getenv("GITHUB_CLIENT_ID"),
		os.Getenv("GITHUB_CLIENT_SECRET"),
		cookiemanager,
	)

	http.HandleFunc("/", handleTop)
	http.HandleFunc("/check", handleCheck)
	http.HandleFunc("/webhook", handleWebhook)
	http.HandleFunc("/gh/_auth/callback", ghHandler.HandleAuthCallback)
	http.Handle("/gh/", ghHandler.Handler(http.HandlerFunc(ghHandler.HandleGitHubTop)))
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
