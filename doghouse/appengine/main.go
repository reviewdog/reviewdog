package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/haya14busa/secretbox"
	"github.com/justinas/nosurf"
	"github.com/reviewdog/reviewdog/doghouse/server/cookieman"
	"github.com/reviewdog/reviewdog/doghouse/server/storage"
	"google.golang.org/appengine"
)

func mustCookieMan() *cookieman.CookieMan {
	// Create secret key by following command.
	// $ ruby -rsecurerandom -e 'puts SecureRandom.hex(32)'
	cipher, err := secretbox.NewFromHexKey(mustGetenv("SECRETBOX_SECRET"))
	if err != nil {
		log.Fatalf("failed to create secretbox: %v", err)
	}
	c := cookieman.CookieOption{
		Cookie: http.Cookie{
			HttpOnly: true,
			Secure:   !appengine.IsDevAppServer(),
			Path:     "/",
		},
	}
	return cookieman.New(cipher, c)
}

func mustGitHubAppsPrivateKey() []byte {
	// Private keys https://github.com/settings/apps/reviewdog
	githubAppsPrivateKey, err := ioutil.ReadFile(mustGetenv("GITHUB_PRIVATE_KEY_FILE"))
	if err != nil {
		log.Fatalf("could not read private key: %s", err)
	}
	return githubAppsPrivateKey
}

func mustGetenv(name string) string {
	s := os.Getenv(name)
	if s == "" {
		log.Fatalf("%s is not set", name)
	}
	return s
}

func mustIntEnv(name string) int {
	s := os.Getenv(name)
	if s == "" {
		log.Fatalf("%s is not set", name)
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatal(err)
	}
	return i
}

func main() {
	initTemplates()

	integrationID := mustIntEnv("GITHUB_INTEGRATION_ID")
	ghPrivateKey := mustGitHubAppsPrivateKey()

	ghInstStore := storage.GitHubInstallationDatastore{}
	ghRepoTokenStore := storage.GitHubRepoTokenDatastore{}

	ghHandler := NewGitHubHandler(
		mustGetenv("GITHUB_CLIENT_ID"),
		mustGetenv("GITHUB_CLIENT_SECRET"),
		mustCookieMan(),
		ghPrivateKey,
		integrationID,
	)

	ghChecker := githubChecker{
		privateKey:       ghPrivateKey,
		integrationID:    integrationID,
		ghInstStore:      &ghInstStore,
		ghRepoTokenStore: &ghRepoTokenStore,
	}

	ghWebhookHandler := githubWebhookHandler{
		secret:      []byte(mustGetenv("GITHUB_WEBHOOK_SECRET")),
		ghInstStore: &ghInstStore,
	}

	mu := http.NewServeMux()

	// Register Admin handlers.
	mu.HandleFunc("/_ah/warmup", warmupHandler)

	mu.HandleFunc("/", handleTop)
	mu.HandleFunc("/check", ghChecker.handleCheck)
	mu.HandleFunc("/gh_/webhook", ghWebhookHandler.handleWebhook)
	mu.HandleFunc("/gh_/auth/callback", ghHandler.HandleAuthCallback)
	mu.HandleFunc("/gh_/logout", ghHandler.HandleLogout)
	mu.Handle("/gh/", nosurf.New(ghHandler.LogInHandler(http.HandlerFunc(ghHandler.HandleGitHubTop))))

	http.Handle("/", mu)
	appengine.Main()
}

func handleTop(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Title string
	}
	data.Title = "reviewdog"
	topTmpl.ExecuteTemplate(w, "base", &data)
}
