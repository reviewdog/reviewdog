# Working with reviewdog

## What This Project Does

reviewdog is a command-line tool that takes linter/compiler output from stdin, parses it into a universal diagnostic format, filters results by diff (only showing issues on changed lines), and posts them as review comments on GitHub, GitLab, Bitbucket, Gerrit, or Gitea.

## Build and Test

```bash
# Build
go install ./cmd/reviewdog

# Run all tests with race detector
go test -v -race ./...

# Run tests for a specific package
go test -v ./parser/
go test -v ./filter/
go test -v ./service/github/

# Run tests with coverage (matches CI)
go test -v -race -coverpkg=./... -coverprofile=coverage.txt ./...
```

## Architecture: The Pipeline

The core flow is: **Parse -> Diff -> Filter -> Report**.

1. **Parse** (`parser/`): Linter output is parsed into `[]*rdf.Diagnostic`. The `parser.New()` factory routes by format name: `errorformat`, `checkstyle`, `rdjson`, `rdjsonl`, `sarif`, `diff`.

2. **Diff** (`diff/`): Unified diff is parsed into `[]FileDiff`. The `DiffService` interface provides diff data -- either from a shell command (`DiffCmd`) or from a platform API (e.g. GitHub PR diff).

3. **Filter** (`filter/`): `FilterCheck()` matches diagnostics against diff hunks. Four modes: `added` (default, only new lines), `diff_context`, `file`, `nofilter`. Output is `[]*FilteredDiagnostic` with `ShouldReport`, `InDiffFile`, `InDiffContext` fields.

4. **Report** (`service/*/`): `CommentService.Post()` sends filtered results to the platform. `BulkCommentService.Flush()` handles batched posting. GitHub has three reporter types: PR review comments, Check Runs API, and Actions log annotations.

## Key Interfaces

These three interfaces define the entire plugin architecture:

```go
// reviewdog.go
type CommentService interface {
    Post(context.Context, *Comment) error
    ShouldPrependGitRelDir() bool
}

type DiffService interface {
    Diff(context.Context) ([]byte, error)
    Strip() int
}

// parser/parser.go
type Parser interface {
    Parse(r io.Reader) ([]*rdf.Diagnostic, error)
}
```

## The Reviewdog Diagnostic Format (RDF)

Defined in `proto/rdf/reviewdog.proto`. This is the universal interchange format. Key types:

- `Diagnostic`: message, location, severity, source, code, suggestions
- `Location`: path + Range
- `Range`: start Position (inclusive) to end Position (exclusive)
- `Suggestion`: range + replacement text (supports insert, update, delete)
- `Severity`: UNKNOWN_SEVERITY, ERROR, WARNING, INFO

Wire formats: `rdjsonl` (JSON Lines of Diagnostic) and `rdjson` (JSON of DiagnosticResult).

JSON schemas are in `proto/rdf/jsonschema/`.

## Adding a New Input Format Parser

1. Create `parser/yourformat.go` implementing `Parser` interface
2. Add the format name to the switch in `parser.New()` (`parser/parser.go`)
3. Create `parser/yourformat_test.go` with test cases
4. Add test fixtures to `_testdata/` if needed

## Adding a New Platform Reporter

1. Create `service/yourplatform/` package
2. Implement `CommentService` (and optionally `BulkCommentService`)
3. Implement `DiffService` for fetching PR/MR diffs from the platform API
4. Wire it into the CLI in `cmd/reviewdog/main.go` under the reporter flag
5. Add the reporter name to the `reporterDoc` constant
6. Add CI environment variable detection in `cienv/` if needed

## Comment Deduplication (MetaComment)

reviewdog embeds base64-encoded metadata in review comments (`proto/metacomment/metacomment.proto`) containing a fingerprint (content hash) and source name (tool name). On subsequent runs, it reads existing comments, extracts metadata, and skips already-posted diagnostics. It also cleans up outdated comments from the same tool.

## Project Config (.reviewdog.yml)

The `project/` package handles the `.reviewdog.yml` config file, which defines multiple linter runners:

```yaml
runner:
  golint:
    cmd: golint ./...
    format: golint
    level: warning
  govet:
    cmd: go vet ./...
    format: govet
```

Runners execute in parallel (bounded by `runtime.NumCPU()`). Secret env vars (`REVIEWDOG_GITHUB_API_TOKEN`, etc.) are stripped from runner subprocesses.

## Doghouse Server

The `doghouse/` directory contains a server component (deployed on App Engine) that proxies GitHub Checks API calls using GitHub App installation tokens. The CLI communicates with it via `doghouse/client/`. When `REVIEWDOG_SKIP_DOGHOUSE=true` or running in GitHub Actions with `GITHUB_TOKEN`, the CLI talks to GitHub directly.

## CLI Structure

The CLI uses the standard library `flag` package (not cobra). All flags are in `cmd/reviewdog/main.go`. Key flags:

- `-f`: input format name (e.g. `golint`, `checkstyle`, `rdjson`, `rdjsonl`, `sarif`)
- `-efm`: errorformat string (can be repeated)
- `-diff`: diff command for local mode (e.g. `"git diff"`)
- `-reporter`: output target (`local`, `github-check`, `github-pr-review`, `github-pr-check`, `github-annotations`, `gitlab-mr-discussion`, etc.)
- `-filter-mode`: `added`, `diff_context`, `file`, `nofilter`
- `-fail-level`: exit 1 on findings at this severity or above (`none`, `any`, `info`, `warning`, `error`)
- `-conf`: path to `.reviewdog.yml` config file
- `-name`: tool name shown in review comments
- `-tee`: pass-through mode (output linter results to stdout while also reporting)

## Testing Conventions

- Tests use standard `testing` package, `testify/assert`, and `go-cmp` (with `protocmp` for protobuf comparison)
- Interface compliance checks: `var _ Interface = (*Impl)(nil)`
- Test data uses inline strings for diff/lint content rather than external files where practical
- External test fixtures go in `_testdata/`
- CI runs tests with `-race` flag -- all code must be race-condition safe

## Code Conventions

- Error wrapping: `fmt.Errorf("context: %w", err)`
- Concurrency: `sync.Map` for `ResultMap`, `errgroup.Group` for parallel execution, `sync.Mutex` for comment batching
- Protobuf code generation: run `proto/update.sh`
- No `panic()` for recoverable errors
- Package names are single lowercase words
- Interfaces named after their primary role (`CommentService`, not `ICommentService`)

## Release Process

1. Update `CHANGELOG.md`
2. Create a PR with label `bump:patch`, `bump:minor`, or `bump:major`
3. On merge, `haya14busa/action-bumpr` auto-tags and GoReleaser creates releases

## Important: Things to Watch Out For

- GitHub API rate limits: review comments are capped at 30 per request (`maxCommentsPerRequest`), check annotations at 50 (`maxAnnotationsPerRequest`)
- Path normalization is critical: `pathutil.NormalizePath()` handles relative paths, `./` prefixes, and git working directory offsets
- The `ShouldPrependGitRelDir()` method matters for correctness when running in subdirectories
- Filter mode `added` is the safe default; `nofilter` can be very noisy
- Secret env vars are intentionally stripped from runner subprocesses in project mode
- GitHub Actions fallback: when the token lacks review permission (e.g. fork PRs), GitHub service falls back to Actions log annotations
