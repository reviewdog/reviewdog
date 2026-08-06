package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v90/github"
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
	cli, err := githubClient(token)
	if err != nil {
		return err
	}
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
		if repo.GetArchived() || !strings.HasPrefix(repo.GetName(), "action-") {
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

func githubClient(token string) (*github.Client, error) {
	return github.NewClient(github.WithAuthToken(token))
}
