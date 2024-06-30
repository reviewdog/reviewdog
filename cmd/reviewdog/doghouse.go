package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/cienv"
	"github.com/reviewdog/reviewdog/doghouse"
	"github.com/reviewdog/reviewdog/doghouse/client"
	"github.com/reviewdog/reviewdog/pathutil"
	"github.com/reviewdog/reviewdog/project"
	"github.com/reviewdog/reviewdog/proto/rdf"
	"github.com/reviewdog/reviewdog/service/serviceutil"
)

func runDoghouse(ctx context.Context, r io.Reader, w io.Writer, opt *option, isProject bool) error {
	ghInfo, _, err := cienv.GetBuildInfo()
	if err != nil {
		return err
	}
	resultSet, err := checkResultSet(ctx, r, opt, isProject)
	if err != nil {
		return err
	}
	cli := newDoghouseCli(ctx)
	if cli == nil {
		return errors.New("failed to create a doghouse client")
	}
	if err := postResultSet(ctx, resultSet, ghInfo, cli, opt); err != nil {
		return err
	}
	return nil
}

// If skipDoghouseServer is true, reviewdog won't talk to the doghouse server
// because provided GitHub API Token has Check API scope.
// You can force skipping the doghouse server if you are generating your own
// application API token.
func skipDoghouseServer() bool {
	return (os.Getenv("REVIEWDOG_SKIP_DOGHOUSE") == "true" || cienv.IsInGitHubAction()) && os.Getenv("REVIEWDOG_TOKEN") == ""
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
		resultSet, err = projectRunAndParse(ctx, conf, buildRunnersMap(opt.runners), opt.level, opt.tee)
		if err != nil {
			return nil, err
		}
	} else {
		p, err := newParserFromOpt(opt)
		if err != nil {
			return nil, err
		}
		diagnostics, err := p.Parse(r)
		if err != nil {
			return nil, err
		}
		resultSet.Store(toolName(opt), &reviewdog.Result{
			Level:       opt.level,
			Diagnostics: diagnostics,
		})
	}
	return resultSet, nil
}

func postResultSet(ctx context.Context, resultSet *reviewdog.ResultMap,
	ghInfo *cienv.BuildInfo, cli *client.DogHouseClient, opt *option) error {
	var g errgroup.Group
	wd, _ := os.Getwd()
	gitRelWd, err := serviceutil.GitRelWorkdir()
	if err != nil {
		return err
	}
	resultSet.Range(func(name string, result *reviewdog.Result) {
		diagnostics := result.Diagnostics
		as := make([]*doghouse.Annotation, 0, len(diagnostics))
		for _, d := range diagnostics {
			as = append(as, checkResultToAnnotation(d, wd, gitRelWd))
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
			FilterMode:  opt.filterMode,
		}
		g.Go(func() error {
			if err := result.CheckUnexpectedFailure(); err != nil {
				return err
			}

			res, err := cli.Check(ctx, req)

			if err != nil {
				return fmt.Errorf("post failed for %s: %w", name, err)
			}
			if res.ReportURL != "" {
				conclusion := ""
				if res.Conclusion != "" {
					conclusion = fmt.Sprintf(" (conclusion=%s)", res.Conclusion)
				}
				log.Printf("[%s] reported: %s%s", name, res.ReportURL, conclusion)
			}
			if res.ReportURL == "" {
				return fmt.Errorf("[%s] no result found", name)
			}
			// If failOnError is on, return error when at least one report
			// returns failure conclusion (status). Users can check this
			// reviewdog run status (#446) to merge PRs for example.
			//
			// Also, the individual report conclusions are associated to random check
			// suite due to the GitHub bug (#403), so actually users cannot depends
			// on each report as of writing.
			if opt.failOnError && (res.Conclusion == "failure") {
				return fmt.Errorf("[%s] Check conclusion is %q", name, res.Conclusion)
			}
			return nil
		})
	})
	return g.Wait()
}

func checkResultToAnnotation(d *rdf.Diagnostic, wd, gitRelWd string) *doghouse.Annotation {
	d.GetLocation().Path = pathutil.NormalizePath(d.GetLocation().GetPath(), wd, gitRelWd)
	return &doghouse.Annotation{
		Diagnostic: d,
	}
}
