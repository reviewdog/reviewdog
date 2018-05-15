package main

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/haya14busa/reviewdog"
	"github.com/haya14busa/reviewdog/doghouse"
	"github.com/haya14busa/reviewdog/doghouse/client"
	"github.com/haya14busa/reviewdog/project"
	"golang.org/x/sync/errgroup"
)

func runDoghouse(ctx context.Context, r io.Reader, opt *option, isProject bool) error {
	ghInfo, isPr, err := getGitHubPR(opt.ci)
	if err != nil {
		return err
	}
	if !isPr {
		return errors.New("this is not PullRequest build.")
	}

	resultSet := make(map[string][]*reviewdog.CheckResult)

	if isProject {
		conf, err := projectConfig(opt.conf)
		if err != nil {
			return err
		}
		resultSet, err = project.RunAndParse(ctx, conf)
		if err != nil {
			return err
		}
	} else {
		p, err := newParserFromOpt(opt)
		if err != nil {
			return err
		}
		rs, err := p.Parse(r)
		if err != nil {
			return err
		}
		resultSet[toolName(opt)] = rs
	}

	cli := client.New(nil)
	var g errgroup.Group
	wd, _ := os.Getwd()
	for name, results := range resultSet {
		name := name
		as := make([]*doghouse.Annotation, 0, len(results))
		for _, r := range results {
			as = append(as, checkResultToAnnotation(r, wd))
		}
		req := &doghouse.CheckRequest{
			Name:        name,
			Owner:       ghInfo.owner,
			Repo:        ghInfo.repo,
			PullRequest: ghInfo.pr,
			SHA:         ghInfo.sha,
			Annotations: as,
		}
		if id := installationID(); id != 0 {
			req.InstallationID = id
		}
		g.Go(func() error {
			res, err := cli.Check(ctx, req)
			if err != nil {
				return err
			}
			log.Printf("[%s] reported: %s", name, res.ReportURL)
			return nil
		})
	}
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

func installationID() int {
	id, _ := strconv.Atoi(os.Getenv("REVIEWDOG_GITHUB_APP_INSTALLATION_ID"))
	return id
}
