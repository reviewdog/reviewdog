# Copilot Instructions for reviewdog

## Project Overview

reviewdog is a Go CLI tool that pipes linter/compiler output through a parse-filter-report pipeline to post review comments on GitHub, GitLab, Bitbucket, Gerrit, and Gitea. It only shows issues on changed lines (diff-filtered).

## Code Style

- Standard Go conventions. Use `gofmt` formatting.
- Error wrapping with `fmt.Errorf("context: %w", err)`. Never bare `return err` without context at package boundaries.
- No `panic()` for recoverable errors.
- Interfaces named by role: `CommentService`, `DiffService`, `Parser`. No `I` prefix.
- Package names are single lowercase words: `parser`, `filter`, `diff`, `cienv`.
- CLI uses standard library `flag` package, not cobra.

## Architecture Awareness

The core pipeline is: **Parse -> Diff -> Filter -> Report**.

Three key interfaces drive the entire system:

```go
// reviewdog.go - implement these to add platform support
type CommentService interface {
    Post(context.Context, *Comment) error
    ShouldPrependGitRelDir() bool
}

type DiffService interface {
    Diff(context.Context) ([]byte, error)
    Strip() int
}

// parser/parser.go - implement this to add format support
type Parser interface {
    Parse(r io.Reader) ([]*rdf.Diagnostic, error)
}
```

All diagnostic data flows through `*rdf.Diagnostic` (defined in `proto/rdf/reviewdog.proto`). This is the universal interchange format.

## Directory Layout

- `cmd/reviewdog/` - CLI entry point, flag parsing, reporter wiring
- `parser/` - Input format parsers (errorformat, checkstyle, rdjson, rdjsonl, sarif, diff)
- `filter/` - Diff-based result filtering (added, diff_context, file, nofilter modes)
- `diff/` - Unified diff parser
- `proto/rdf/` - Reviewdog Diagnostic Format protobuf and JSON schemas
- `proto/metacomment/` - Comment deduplication metadata proto
- `service/github/` - GitHub reporters (PR review, Check Runs, Actions annotations)
- `service/gitlab/` - GitLab MR reporter
- `service/bitbucket/` - Bitbucket Code Insights reporter
- `service/gerrit/` - Gerrit change review reporter
- `service/gitea/` - Gitea PR review reporter
- `service/commentutil/` - Shared markdown formatting helpers
- `service/serviceutil/` - Fingerprinting, git root detection, metacomment encoding
- `doghouse/` - Server proxy for GitHub Checks API (App Engine deployment)
- `cienv/` - CI environment auto-detection (GitHub Actions, Travis, Circle, Drone, GitLab CI, etc.)
- `project/` - `.reviewdog.yml` config parsing and multi-runner orchestration
- `pathutil/` - Path normalization (critical for correct file matching)
- `_testdata/` - Test fixtures

## Testing Patterns

- Tests live alongside source (`*_test.go`), not in separate directories.
- Interface compliance: `var _ CommentService = (*PullRequest)(nil)`
- Use `github.com/google/go-cmp/cmp` with `protocmp.Transform()` for protobuf comparison.
- Use `github.com/stretchr/testify` for assertions.
- Inline diff/lint strings in test functions rather than external files where practical.
- Test fixtures in `_testdata/` when external files are needed.
- All code must pass `-race` flag. Use proper synchronization.

## Concurrency Patterns

- `sync.Map` for `ResultMap` / `FilteredResultMap` (concurrent linter results).
- `errgroup.Group` for parallel linter execution and parallel API calls.
- `sync.Mutex` for comment batching in service implementations.
- Semaphore pattern via buffered channel for bounding parallelism in project runner.

## Key Constants and Limits

- `maxCommentsPerRequest = 30` (GitHub review comments per request)
- `maxFileComments = 10` (GitHub file-level comments per flush)
- `maxAnnotationsPerRequest = 50` (GitHub Check API limit)
- `maxAllowedSize = 65535` (GitHub Check run summary character limit)

## When Generating New Code

- New parsers go in `parser/` and must implement `Parser` interface. Register in `parser.New()` switch.
- New reporters go in `service/<platform>/` and must implement `CommentService`. Wire into `cmd/reviewdog/main.go`.
- Use `rdf.Diagnostic` as the universal data structure, never create parallel types.
- Path handling must go through `pathutil.NormalizePath()` for correctness.
- Secret env vars (`REVIEWDOG_GITHUB_API_TOKEN`, `REVIEWDOG_GITLAB_API_TOKEN`, `REVIEWDOG_TOKEN`) are stripped from runner subprocesses -- respect this pattern.
- Comment deduplication uses `serviceutil.Fingerprint()` and `serviceutil.BuildMetaComment()` -- use these in new reporters.

## Build and CI

```bash
go test -v -race ./...         # Run tests
go install ./cmd/reviewdog     # Build CLI
```

CI runs on GitHub Actions. The project dogfoods itself: `reviewdog.yml` runs reviewdog on its own PRs with multiple linters (golint, golangci-lint, staticcheck, misspell, typos, shellcheck, etc.).

Releases use `haya14busa/action-bumpr` with PR labels (`bump:patch`, `bump:minor`, `bump:major`) and GoReleaser.
