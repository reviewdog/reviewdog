package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v38/github"
	"golang.org/x/oauth2"
)

var (
	targetOrg = flag.String("org", "reviewdog", "target org name")
)

func main() {
	flag.Parse()
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	token := os.Getenv("DEPUP_GITHUB_API_TOKEN")
	if token == "" {
		return errors.New("DEPUP_GITHUB_API_TOKEN is empty")
	}
	cli := githubClient(ctx, token)
	// TODO(haya14busa): Support pagination once the # of repo become more than 100.
	repos, _, err := cli.Repositories.ListByOrg(ctx, *targetOrg, &github.RepositoryListByOrgOptions{
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	})
	if err != nil {
		return err
	}
	var wholeErr error
	for _, repo := range repos {
		if !strings.HasPrefix(repo.GetName(), "action-") {
			continue
		}
		log.Printf("Dispatch depup to %s/%s...", *targetOrg, repo.GetName())
		if _, _, err := cli.Repositories.Dispatch(ctx, *targetOrg, repo.GetName(), github.DispatchRequestOptions{
			EventType: "depup",
		}); err != nil {
			log.Printf("Dispatch depup to %s/%s failed: %v", *targetOrg, repo.GetName(), err)
			wholeErr = err
		}
	}
	return wholeErr
}

func githubClient(ctx context.Context, token string) *github.Client {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{})
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}
