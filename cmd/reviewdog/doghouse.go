package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/cienv"
	"github.com/reviewdog/reviewdog/doghouse"
	"github.com/reviewdog/reviewdog/doghouse/client"
	"github.com/reviewdog/reviewdog/project"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

// If skipDoghouseServer is true, run doghouse code directly instead of talking to
// the doghouse server.
func runDoghouse(ctx context.Context, r io.Reader, opt *option, isProject bool, skipDoghouseServer bool) error {
	ghInfo, isPr, err := cienv.GetBuildInfo()
	if err != nil {
		return err
	}
	if !isPr {
		fmt.Fprintln(os.Stderr, "reviewdog: this is not PullRequest build.")
		return nil
	}
	resultSet, err := checkResultSet(ctx, r, opt, isProject)
	if err != nil {
		return err
	}
	var cli client.DogHouseClientInterface
	if skipDoghouseServer {
		token, err := nonEmptyEnv("REVIEWDOG_GITHUB_API_TOKEN")
		if err != nil {
			return err
		}
		ghcli, err := githubClient(ctx, token)
		if err != nil {
			return err
		}
		log.Println("[debug] use GitHub client for Doghouse run")
		cli = &client.GitHubClient{Client: ghcli}
	} else {
		cli = newDoghouseCli(ctx)
	}
	return postResultSet(ctx, resultSet, ghInfo, cli)
}

func newDoghouseCli(ctx context.Context) *client.DogHouseClient {
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
		resultSet, err = projectRunAndParse(ctx, conf)
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
		resultSet.Store(toolName(opt), rs)
	}
	return resultSet, nil
}

func postResultSet(ctx context.Context, resultSet *reviewdog.ResultMap, ghInfo *cienv.BuildInfo, cli client.DogHouseClientInterface) error {
	var g errgroup.Group
	wd, _ := os.Getwd()
	resultSet.Range(func(name string, results []*reviewdog.CheckResult) {
		log.Println("[debug] post: " + name)
		as := make([]*doghouse.Annotation, 0, len(results))
		for _, r := range results {
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
		}
		g.Go(func() error {
			res, err := cli.Check(ctx, req)
			if err != nil {
				return fmt.Errorf("post failed for %s: %v", name, err)
			}
			log.Printf("[%s] reported: %s", name, res.ReportURL)
			return nil
		})
	})
	return g.Wait()
}

func checkResultToAnnotation(c *reviewdog.CheckResult, wd string) *doghouse.Annotation {
	return &doghouse.Annotation{
		Path:       reviewdog.CleanPath(c.Path, wd),
		Line:       c.Lnum,
		Message:    c.Message,
		RawMessage: strings.Join(c.Lines, "\n"),
	}
}
