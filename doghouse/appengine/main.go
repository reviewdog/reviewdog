package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"github.com/haya14busa/secretbox"
	"github.com/justinas/nosurf"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"

	"github.com/reviewdog/reviewdog/doghouse/server/cookieman"
	"github.com/reviewdog/reviewdog/doghouse/server/storage"
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
			Secure:   true,
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
	configureTrace()
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
		tr: &ochttp.Transport{
			// Use Google Cloud propagation format.
			Propagation: &propagation.HTTPFormat{},
		},
	}

	ghWebhookHandler := githubWebhookHandler{
		secret:      []byte(mustGetenv("GITHUB_WEBHOOK_SECRET")),
		ghInstStore: &ghInstStore,
	}

	mu := http.NewServeMux()

	// Register Admin handlers.
	mu.HandleFunc("/_ah/warmup", warmupHandler)

	handleFunc(mu, "/", handleTop)
	handleFunc(mu, "/check", ghChecker.handleCheck)
	handleFunc(mu, "/gh_/webhook", ghWebhookHandler.handleWebhook)
	handleFunc(mu, "/gh_/auth/callback", ghHandler.HandleAuthCallback)
	handleFunc(mu, "/gh_/logout", ghHandler.HandleLogout)
	mu.Handle("/gh/", nosurf.New(ochttp.WithRouteTag(ghHandler.LogInHandler(http.HandlerFunc(ghHandler.HandleGitHubTop)), "/gh/")))

	http.Handle("/", mu)
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), &ochttp.Handler{
		Handler:     mu,
		Propagation: &propagation.HTTPFormat{},
	}))
}

func handleFunc(mu *http.ServeMux, pattern string, handler func(http.ResponseWriter, *http.Request)) {
	mu.Handle(pattern,
		ochttp.WithRouteTag(http.HandlerFunc(handler), pattern))
}

func handleTop(w http.ResponseWriter, _ *http.Request) {
	var data struct {
		Title string
	}
	data.Title = "reviewdog"
	topTmpl.ExecuteTemplate(w, "base", &data)
}

// Document: https://cloud.google.com/trace/docs/setup/go
func configureTrace() {
	// Create and register a OpenCensus Stackdriver Trace exporter.
	exporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: os.Getenv("GOOGLE_CLOUD_PROJECT"),
	})
	if err != nil {
		log.Fatal(err)
	}
	trace.RegisterExporter(exporter)
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
}
