package github

import (
	"context"
	"path/filepath"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/service/github/githubutils"
	"github.com/reviewdog/reviewdog/service/serviceutil"
)

var _ reviewdog.CommentService = (*Log)(nil)

// Log is a logging service for GitHub Actions.
type Log struct {
	logWriter *githubutils.GitHubActionLogWriter

	// wd is working directory relative to root of repository.
	wd string
}

// NewGitHubActionLog returns a new Log service.
func NewGitHubActionLog(level string) (*Log, error) {
	workDir, err := serviceutil.GitRelWorkdir()
	if err != nil {
		return nil, err
	}
	return &Log{
		logWriter: githubutils.NewGitHubActionLogWriter(level),
		wd:        workDir,
	}, nil
}

// Post writes a log.
func (g *Log) Post(ctx context.Context, c *reviewdog.Comment) error {
	c.Result.Diagnostic.GetLocation().Path = filepath.ToSlash(filepath.Join(g.wd,
		c.Result.Diagnostic.GetLocation().GetPath()))
	return g.logWriter.Post(ctx, c)
}

// Flush checks overall error at last.
func (g *Log) Flush(ctx context.Context) error {
	return g.logWriter.Flush(ctx)
}
