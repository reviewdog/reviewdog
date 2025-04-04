# Reviewdog Development Guide

## Build Commands
- Build: `go build ./cmd/reviewdog`
- Run tests: `go test -v ./...`
- Run specific test: `go test -v github.com/reviewdog/reviewdog/[package] -run [TestName]`
- Test with race detection: `go test -v -race ./...`
- Coverage report: `go test -v -coverpkg=./... -coverprofile=coverage.txt ./...`

## Code Style
- Follow standard Go conventions (gofmt)
- Error handling: Always check errors and return them with context using `fmt.Errorf("message: %w", err)`
- Interfaces are preferred for extensibility and testing
- Context should be passed to functions that perform I/O or network operations
- Use meaningful variable/function names that describe intent

## Import Conventions
- Stdlib imports first, then external packages, then internal packages
- Use full imports paths (no dot imports)
- Group related imports together

## Documentation
- All exported functions, types, and variables must be documented
- Include examples for public APIs where appropriate

## Testing
- All packages should have tests
- Test both success and error cases
- Use table-driven tests where appropriate