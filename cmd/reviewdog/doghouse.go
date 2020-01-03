package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/cienv"
	"github.com/reviewdog/reviewdog/doghouse"
	"github.com/reviewdog/reviewdog/doghouse/client"
	"github.com/reviewdog/reviewdog/project"
)

func runDoghouse(ctx context.Context, r io.Reader, w io.Writer, opt *option, isProject bool, forPr bool) error {
	ghInfo, isPr, err := cienv.GetBuildInfo()
	if err != nil {
		return err
	}
	if !isPr && forPr {
		fmt.Fprintln(os.Stderr, "reviewdog: this is not PullRequest build.")
		return nil
	}
	if !forPr {
		ghInfo.PullRequest = 0
	}
	resultSet, err := checkResultSet(ctx, r, opt, isProject)
	if err != nil {
		return err
	}
	cli, err := newDoghouseCli(ctx)
	if err != nil {
		return err
	}
	filteredResultSet, err := postResultSet(ctx, resultSet, ghInfo, cli)
	if err != nil {
		return err
	}
	if foundResultInDiff := reportResults(w, filteredResultSet); foundResultInDiff {
		return errors.New("found at least one result in diff")
	}
	return nil
}

func newDoghouseCli(ctx context.Context) (client.DogHouseClientInterface, error) {
	// If skipDoghouseServer is true, run doghouse code directly instead of talking to
	// the doghouse server because provided GitHub API Token has Check API scope.
	skipDoghouseServer := cienv.IsInGitHubAction() && os.Getenv("REVIEWDOG_TOKEN") == ""
	if skipDoghouseServer {
		token, err := nonEmptyEnv("REVIEWDOG_GITHUB_API_TOKEN")
		if err != nil {
			return nil, err
		}
		ghcli, err := githubClient(ctx, token)
		if err != nil {
			return nil, err
		}
		return &client.GitHubClient{Client: ghcli}, nil
	}
	return newDoghouseServerCli(ctx), nil
}

func newDoghouseServerCli(ctx context.Context) *client.DogHouseClient {
	httpCli := http.DefaultClient
	if token := os.Getenv("REVIEWDOG_TOKEN"); token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		httpCli = oauth2.NewClient(ctx, ts)
	}
	return client.New(httpCli)
}

var projectRunAndParse = project.RunAndParse

func checkResultSet(ctx context.Context, r io.Reader, opt *option, isProject bool) (*reviewdog.ResultMap, error) {
	resultSet := new(reviewdog.ResultMap)
	if isProject {
		conf, err := projectConfig(opt.conf)
		if err != nil {
			return nil, err
		}
		resultSet, err = projectRunAndParse(ctx, conf, buildRunnersMap(opt.runners), opt.level, opt.tee)
		if err != nil {
			return nil, err
		}
	} else {
		p, err := newParserFromOpt(opt)
		if err != nil {
			return nil, err
		}
		rs, err := p.Parse(r)
		if err != nil {
			return nil, err
		}
		resultSet.Store(toolName(opt), &reviewdog.Result{
			Level:        opt.level,
			CheckResults: rs,
		})
	}
	return resultSet, nil
}

func postResultSet(ctx context.Context, resultSet *reviewdog.ResultMap, ghInfo *cienv.BuildInfo, cli client.DogHouseClientInterface) (*reviewdog.FilteredCheckMap, error) {
	var g errgroup.Group
	wd, _ := os.Getwd()
	filteredResultSet := new(reviewdog.FilteredCheckMap)
	resultSet.Range(func(name string, result *reviewdog.Result) {
		checkResults := result.CheckResults
		as := make([]*doghouse.Annotation, 0, len(checkResults))
		for _, r := range checkResults {
			as = append(as, checkResultToAnnotation(r, wd))
		}
		req := &doghouse.CheckRequest{
			Name:        name,
			Owner:       ghInfo.Owner,
			Repo:        ghInfo.Repo,
			PullRequest: ghInfo.PullRequest,
			SHA:         ghInfo.SHA,
			Branch:      ghInfo.Branch,
			Annotations: as,
			Level:       result.Level,
		}
		g.Go(func() error {
			res, err := cli.Check(ctx, req)
			if err != nil {
				return fmt.Errorf("post failed for %s: %v", name, err)
			}
			if res.ReportURL != "" {
				log.Printf("[%s] reported: %s", name, res.ReportURL)
			}
			if res.CheckedResults != nil {
				filteredResultSet.Store(name, res.CheckedResults)
			}
			if res.ReportURL == "" && res.CheckedResults == nil {
				return fmt.Errorf("no result found for %q", name)
			}
			return nil
		})
	})
	return filteredResultSet, g.Wait()
}

func checkResultToAnnotation(c *reviewdog.CheckResult, wd string) *doghouse.Annotation {
	return &doghouse.Annotation{
		Path:       reviewdog.CleanPath(c.Path, wd),
		Line:       c.Lnum,
		Message:    c.Message,
		RawMessage: strings.Join(c.Lines, "\n"),
	}
}

// reportResults reports results to given io.Writer and return true if at least
// one annotation result is in diff.
func reportResults(w io.Writer, filteredResultSet *reviewdog.FilteredCheckMap) bool {
	// Sort names to get deterministic result.
	var names []string
	filteredResultSet.Range(func(name string, results []*reviewdog.FilteredCheck) {
		names = append(names, name)
	})
	sort.Strings(names)

	foundInDiff := false
	for _, name := range names {
		results, err := filteredResultSet.Load(name)
		if err != nil {
			// Should not happen.
			log.Printf("reviewdog: result not found for %q", name)
			continue
		}
		fmt.Fprintf(w, "reviewdog: Reporting results for %q\n", name)
		foundResultPerName := false
		filteredNum := 0
		for _, result := range results {
			if !result.InDiff {
				filteredNum++
				continue
			}
			foundInDiff = true
			foundResultPerName = true
			// Output original lines.
			for _, line := range result.Lines {
				fmt.Fprintln(w, line)
			}
		}
		if !foundResultPerName {
			fmt.Fprintf(w, "reviewdog: No results found for %q. %d results found outside diff.\n", name, filteredNum)
		}
	}
	return foundInDiff
}
