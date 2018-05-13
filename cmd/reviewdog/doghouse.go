package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/haya14busa/reviewdog"
	"github.com/haya14busa/reviewdog/doghouse"
	"github.com/haya14busa/reviewdog/doghouse/client"
	"github.com/haya14busa/reviewdog/project"
	"golang.org/x/sync/errgroup"
)

func runDoghouse(ctx context.Context, r io.Reader, installationID string, opt *option, isProject bool) error {
	id, err := strconv.Atoi(installationID)
	if err != nil {
		return fmt.Errorf("installationID should be integer: %v, got %s", err, installationID)
	}

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
	for name, results := range resultSet {
		name := name
		as := make([]*doghouse.Annotation, 0, len(results))
		for _, r := range results {
			as = append(as, checkResultToAnnotation(r))
		}
		req := &doghouse.CheckRequest{
			InstallationID: id,
			Name:           name,
			Owner:          ghInfo.owner,
			Repo:           ghInfo.repo,
			PullRequest:    ghInfo.pr,
			SHA:            ghInfo.sha,
			Annotations:    as,
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

func checkResultToAnnotation(c *reviewdog.CheckResult) *doghouse.Annotation {
	return &doghouse.Annotation{
		Path:       c.Path,
		Line:       c.Lnum,
		Message:    c.Message,
		RawMessage: strings.Join(c.Lines, "\n"),
	}
}
