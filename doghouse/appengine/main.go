package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/haya14busa/secretbox"
	"github.com/justinas/nosurf"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/oauth"

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
	githubAppsPrivateKey, err := os.ReadFile(mustGetenv("GITHUB_PRIVATE_KEY_FILE"))
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
		tr:               otelhttp.NewTransport(http.DefaultTransport),
	}

	ghWebhookHandler := githubWebhookHandler{
		secret:      []byte(mustGetenv("GITHUB_WEBHOOK_SECRET")),
		ghInstStore: &ghInstStore,
	}

	mu := http.NewServeMux()

	// Register Admin handlers.
	mu.HandleFunc("/_ah/warmup", warmupHandler)

	// otelhttp records the matched ServeMux pattern as the route.
	mu.HandleFunc("/", handleTop)
	mu.HandleFunc("/check", ghChecker.handleCheck)
	mu.HandleFunc("/gh_/webhook", ghWebhookHandler.handleWebhook)
	mu.HandleFunc("/gh_/auth/callback", ghHandler.HandleAuthCallback)
	mu.HandleFunc("/gh_/logout", ghHandler.HandleLogout)
	mu.Handle("/gh/", nosurf.New(ghHandler.LogInHandler(http.HandlerFunc(ghHandler.HandleGitHubTop))))

	http.Handle("/", mu)
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), otelhttp.NewHandler(mu, "doghouse")))
}

func handleTop(w http.ResponseWriter, _ *http.Request) {
	var data struct {
		Title string
	}
	data.Title = "reviewdog"
	topTmpl.ExecuteTemplate(w, "base", &data)
}

// Document: https://cloud.google.com/trace/docs/migrate-to-otlp-endpoints
func configureTrace() {
	ctx := context.Background()

	// Export traces to Cloud Trace via the Telemetry (OTLP) API.
	creds, err := oauth.NewApplicationDefault(ctx)
	if err != nil {
		log.Fatal(err)
	}
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("telemetry.googleapis.com:443"),
		otlptracegrpc.WithDialOption(grpc.WithPerRPCCredentials(creds)),
	)
	if err != nil {
		log.Fatal(err)
	}
	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(
		attribute.String("gcp.project_id", os.Getenv("GOOGLE_CLOUD_PROJECT")),
	))
	if err != nil {
		log.Fatal(err)
	}
	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	))
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}
