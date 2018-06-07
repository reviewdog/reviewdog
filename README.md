## reviewdog - A code review dog who keeps your codebase healthy

[![Gitter](https://badges.gitter.im/haya14busa/reviewdog.svg)](https://gitter.im/haya14busa/reviewdog?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)
[![LICENSE](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/haya14busa/reviewdog)](https://goreportcard.com/report/github.com/haya14busa/reviewdog)
[![GoDoc](https://godoc.org/github.com/haya14busa/reviewdog?status.svg)](https://godoc.org/github.com/haya14busa/reviewdog)
[![releases](https://img.shields.io/github/release/haya14busa/reviewdog.svg)](https://github.com/haya14busa/reviewdog/releases)

![reviewdog logo](https://raw.githubusercontent.com/haya14busa/i/d598ed7dc49fefb0018e422e4c43e5ab8f207a6b/reviewdog/reviewdog.logo.png)

| CI | Status |
| ---- | ----------- |
| [Travis CI](https://travis-ci.org/haya14busa/reviewdog) | [![Travis Build Status][travis-badge]](https://travis-ci.org/haya14busa/reviewdog) |
| [CircleCI](https://circleci.com/gh/haya14busa/reviewdog) | [![CircleCI][circleci-badge]](https://circleci.com/gh/haya14busa/reviewdog) |
| [drone.io](http://droneio.haya14busa.com/haya14busa/reviewdog) | [![drone.io Build Status](http://droneio.haya14busa.com/api/badges/haya14busa/reviewdog/status.svg)](http://droneio.haya14busa.com/haya14busa/reviewdog) |
| [codecov](https://codecov.io/gh/haya14busa/reviewdog) | [![codecov][codecov-badge]](https://codecov.io/gh/haya14busa/reviewdog) |

"reviewdog" provides a way to post review comments to code hosting service,
such as GitHub, automatically by integrating with any linter tools with ease.
It uses any output of lint tools and post them as a comment if the file and
line are in diff of patches to review.

reviewdog also supports run in local environment to filter output of lint tools
by diff.

[design doc](https://docs.google.com/document/d/1mGOX19SSqRowWGbXieBfGPtLnM0BdTkIc9JelTiu6wA/edit?usp=sharing)

[![github-pr-check sample](https://user-images.githubusercontent.com/3797062/40884858-6efd82a0-6756-11e8-9f1a-c6af4f920fb0.png)](https://github.com/haya14busa/reviewdog/pull/131/checks)
![comment in pull-request](https://user-images.githubusercontent.com/3797062/40941822-1d775064-6887-11e8-98e9-4775d37d47f8.png)
![commit status](https://user-images.githubusercontent.com/3797062/40941738-d62acb0a-6886-11e8-858d-7b97aded2a42.png)
[![sample-comment.png](https://raw.githubusercontent.com/haya14busa/i/dc0ccb1e110515ea407c146d99b749018db05c45/reviewdog/sample-comment.png)](https://github.com/haya14busa/reviewdog/pull/24#discussion_r84599728)

Local run

![reviewdog-local-demo.gif](https://raw.githubusercontent.com/haya14busa/i/dc0ccb1e110515ea407c146d99b749018db05c45/reviewdog/reviewdog-local-demo.gif)

### Installation

Get [the binary release](https://github.com/haya14busa/reviewdog/releases) (recommended way)

or

```
go get -u github.com/haya14busa/reviewdog/cmd/reviewdog
```

## Usage

### 'errorformat'

reviewdog accepts any compiler or linter result from stdin and parses it with
scan-f like [**'errorformat'**](https://github.com/haya14busa/errorformat),
which is the port of Vim's [errorformat](http://vimdoc.sourceforge.net/htmldoc/quickfix.html#errorformat)
feature.

For example, if the result format is `{file}:{line number}:{column number}: {message}`,
errorformat should be `%f:%l:%c: %m` and you can pass it as `-efm` arguments.

```
$ golint ./...
comment_iowriter.go:11:6: exported type CommentWriter should have comment or be unexported
$ golint ./... | reviewdog -efm="%f:%l:%c: %m" -diff="git diff master"
```

| name | description |
| ---- | ----------- |
| %f | file name |
| %l | line number |
| %c | column number |
| %m | error message |
| %% | the single '%' character |
| ... | ... |

Please see [haya14busa/errorformat](https://github.com/haya14busa/errorformat)
and [:h errorformat](http://vimdoc.sourceforge.net/htmldoc/quickfix.html#errorformat)
if you want to deal with more complex output. 'errorformat' can handle more
complex output like multi-line error message.

By this 'errorformat' feature, reviewdog can support any tools output with ease.

### Available pre-defined 'errorformat'

But, you don't have to write 'errorformat' in many cases. reviewdog supports
pre-defined errorformat for major compiler or linter tools.

You can find available errorformat name by `reviewdog -list` and you can use it
with `-f={name}`.

```
$ reviewdog -list
golint          linter for Go source code                                       - https://github.com/golang/lint
govet           Vet examines Go source code and reports suspicious problems     - https://golang.org/cmd/vet/
sbt             the interactive build tool                                      - http://www.scala-sbt.org/
...
```

```
$ golint ./... | reviewdog -f=golint -diff="git diff master"
```

You can add supported pre-defined 'errorformat' by contributing to [haya14busa/errorformat](https://github.com/haya14busa/errorformat)

#### checkstyle format

reviewdog also accepts [checkstyle XML format](http://checkstyle.sourceforge.net/) as well.
If the linter supports checkstyle format as a report format, you can us
-f=checkstyle instead of using 'errorformat'.

```
# Local
$ eslint -f checkstyle . | reviewdog -f=checkstyle -diff="git diff"

# CI (overwrite tool name which is shown in review comment by -name arg)
$ eslint -f checkstyle . | reviewdog -f=checkstyle -name="eslint" -reporter=github-pr-check
```

Also, if you want to pass other Json/XML/etc... format to reviewdog, you can write a converter.

```
$ <linter> | <convert-to-checkstyle> | reviewdog -f=checkstyle -name="<linter>" -reporter=github-pr-check
```

### Project Configuration Based Run

reviewdog can also be controlled via the .reviewdog.yml configuration file instead of "-f" or "-efm" arguments.

With .reviewdog.yml, you can run the same commands both CI service and local
environment including editor integration with ease.

#### .reviewdog.yml

```yaml
runner:
  <tool-name>:
    cmd: <command> # (required)
    errorformat: # (optional if there is supporeted format for <tool-name>. see reviewdog -list)
      - <list of errorformat>
    name: <tool-name> # (optional. you can overwrite <tool-name> defined by runner key)

  # examples
  golint:
    cmd: golint ./...
    errorformat:
      - "%f:%l:%c: %m"
  govet:
    cmd: go tool vet -all -shadowstrict .
```

```
$ reviewdog -diff="git diff master"
project/run_test.go:61:28: [golint] error strings should not end with punctuation
project/run.go:57:18: [errcheck]        defer os.Setenv(name, os.Getenv(name))
project/run.go:58:12: [errcheck]        os.Setenv(name, "")
# You can use -conf to specify config file path.
$ reviewdog -conf=./.reviewdog.yml -reporter=github-pr-check
```

Output format for project config based run is one of following formats.

- `<file>: [<tool name>] <message>`
- `<file>:<lnum>: [<tool name>] <message>`
- `<file>:<lnum>:<col>: [<tool name>] <message>`

### Run locally

reviewdog can find new introduced warnings or error by filtering linter results
using diff. You can pass diff command as `-diff` arg, like `-diff="git diff"`,
`-diff="git diff master"`, etc... when you use reviewdog in local environment.

### Working with GitHub and CI services

reviewdog can intergrate with CI service and post review comments to report
results automatically as well!

#### Supported code hosting (or code review) service
At the time of writing, reviewdog supports [GitHub](https://github.com/) and
[GitHub Enterprise](https://enterprise.github.com/home) as a code hosting service for posting comments.

It may support [Gerrit](https://www.gerritcodereview.com/), [Bitbucket](https://bitbucket.org/product), or other services later.

### Reporter: GitHub Checks (-reporter=github-pr-check)
[experimental]

[![github-pr-check sample](https://user-images.githubusercontent.com/3797062/40884858-6efd82a0-6756-11e8-9f1a-c6af4f920fb0.png)](https://github.com/haya14busa/reviewdog/pull/131/checks)

github-pr-review reporter reports results as [GitHub Checks](https://help.github.com/articles/about-status-checks/).
Since GitHub Checks API is only for GitHub Apps, reviewdoc cli send a request to
reviewdog GitHub App server and the server post resutls as GitHub Checks.

1. Install reviewdog Apps. https://github.com/apps/reviewdog
2. Set `REVIEWDOG_TOKEN` or run reviewdog CLI in trusted CI providers.
  - Get token from `https://reviewdog.app/gh/{owner}/{repo-name}`.
  - `$ export REVIEWDOG_TOKEN="xxxxx"`
3. `$ reviewdog -conf=.reviewdog.yml -reporter=github-pr-check`

Note: Token not required if you run reviewdog in Travis or AppVeyor.

#### *Caution:*

As described above, github-pr-review reporter is depending on reviewdog GitHub
App server.
The server is running with haya14busa's pocket money for now and i may break
things, so I cannot ensure that the server is running 24h and 365 days.

github-pr-check reporter is better than github-pr-review reporter in general
because it provides more rich feature and has less scope, but please bear in
mind the above caution and please use it on your own risk.

You can use github-pr-review reporter if you don't want to depens on reviewdog
server.

### Reporter: GitHub PullRequest review comment (-reporter=github-pr-review)

[![sample-comment.png](https://raw.githubusercontent.com/haya14busa/i/dc0ccb1e110515ea407c146d99b749018db05c45/reviewdog/sample-comment.png)](https://github.com/haya14busa/reviewdog/pull/24#discussion_r84599728)

github-pr-review reporter reports results as GitHub PullRequest review comments
using GitHub Personal API Access Token.

1. Get GitHub Personal API Access token
  - Go to https://github.com/settings/tokens and generate new API token.
  - Check `repo` for private repositories or `public_repo` for public repositories.
2. Export the token environment variable
  - `export REVIEWDOG_GITHUB_API_TOKEN="xxxxx"`
3. `$ reviewdog -conf=.reviewdog.yml -reporter=github-pr-review`

```
$ export GITHUB_API="https://example.githubenterprise.com/api/v3/"
$ export REVIEWDOG_INSECURE_SKIP_VERIFY=true # set this as you need
```

For GitHub Enterprise, set API endpoint by environment variable.

```
$ export GITHUB_API="https://example.githubenterprise.com/api/v3/"
$ export REVIEWDOG_INSECURE_SKIP_VERIFY=true # set this as you need
```

### Supported CI services and security of secret environment variable

| Name | Pull Request from the same repository | Pull Request from forked repository |
| ---- | ------------------------------------- | ----------------------------------- |
| [Travis CI](https://travis-ci.org/) | :o: | :x: (:o: for -reporter=github-pr-check)
| [CircleCI](https://circleci.com/) | :o: | :x: (but possible with insecure way)
| [drone.io](https://github.com/drone/drone) (OSS) v0.4 | :o: | :o:
| common (Your managed CI server like Jenkins) | :o: | :o:

reviewdog can run in CI services which supports Pull Request build and secret
environment variable feature for security reason.

But, for the Pull Request from forked repository, most CI services restrict
secret environment variable for security reason, to avoid leak of secret data
with malicious Pull Request. ([Travis CI](https://docs.travis-ci.com/user/pull-requests#Pull-Requests-and-Security-Restrictions), [CircleCI](https://circleci.com/docs/fork-pr-builds/#security-implications-of-running-builds-for-pull-requests-from-forks))


#### Travis CI

##### Travis CI (-reporter=github-pr-check)

If you use -reporter=github-pr-check in Travis CI, you don't need to set `REVIEWDOG_TOKEN`.

Example:

```yaml
env:
  global:
    - REVIEWDOG_VERSION=0.9.9

install:
  - mkdir -p ~/bin/ && export export PATH="~/bin/:$PATH"
  - curl -fSL https://github.com/haya14busa/reviewdog/releases/download/$REVIEWDOG_VERSION/reviewdog_linux_amd64 -o ~/bin/reviewdog && chmod +x ~/bin/reviewdog

script:
  - reviewdog -conf=.reviewdog.yml -reporter=github-pr-check
```

##### Travis CI (-reporter=github-pr-review)

Store GitHub API token by [travis encryption keys](https://docs.travis-ci.com/user/encryption-keys/).

```
$ gem install travis
$ travis encrypt REVIEWDOG_GITHUB_API_TOKEN=<your token> --add env.global
```
Example:

```yaml
env:
  global:
    - secure: xxxxxxxxxxxxx
    - REVIEWDOG_VERSION=0.9.9

install:
  - mkdir -p ~/bin/ && export export PATH="~/bin/:$PATH"
  - curl -fSL https://github.com/haya14busa/reviewdog/releases/download/$REVIEWDOG_VERSION/reviewdog_linux_amd64 -o ~/bin/reviewdog && chmod +x ~/bin/reviewdog

script:
  - >-
    golint ./... | reviewdog -f=golint -reporter=github-pr-review
```

Examples
- https://github.com/azu/textlint-reviewdog-example

#### Circle CI

Store GitHub API token as `REVIEWDOG_TOKEN` or `REVIEWDOG_GITHUB_API_TOKEN` in
[Environment variables - CircleCI](https://circleci.com/docs/environment-variables/#setting-environment-variables-for-all-commands-without-adding-them-to-git)

Pull Requests from fork repo cannot set environment variable for security reason.
However, if you want to run fork PR builds for private repositories or want to run
reviewdog for forked PR to OSS project, CircleCI have an option to enable it.
[Unsafe fork PR builds](https://circleci.com/docs/fork-pr-builds/#unsafe-fork-pr-builds)

If you use -reporter=github-pr-check and only set `REVIEWDOG_TOKEN`, it's safer
because it has only small limited scope. As for `REVIEWDOG_GITHUB_API_TOKEN`,
it has a bit larger scope at least even if you only set the scope repo:public.

Please use it at your own risk.

circle.yml sample

```yaml
version: 2
jobs:
  build:
    docker:
      - image: golang:latest
        environment:
          REVIEWDOG_VERSION: 0.9.9
    steps:
      - checkout
      - run: curl -fSL https://github.com/haya14busa/reviewdog/releases/download/$REVIEWDOG_VERSION/reviewdog_linux_amd64 -o reviewdog && chmod +x ./reviewdog
      - run: go vet ./... 2>&1 | ./reviewdog -f=govet -reporter=github-pr-check
      # or
      - run: go vet ./... 2>&1 | ./reviewdog -f=govet -reporter=github-pr-review
```

#### drone.io
Store GitHub API token in environment variable.  [Secrets · Drone](http://readme.drone.io/usage/secrets/)

Install 'drone' cli command http://readme.drone.io/devs/cli/ and setup configuration.

```
echo '.drone.sec.yaml' >> .gitignore
```

.drone.sec.yaml

```yaml
environment:
  REVIEWDOG_TOKEN: <your token>
  # OR
  REVIEWDOG_GITHUB_API_TOKEN: <your token>
```

.drone.yaml example

```yaml
build:
  lint:
    image: golang
    environment:
      - REVIEWDOG_TOKEN=$$REVIEWDOG_TOKEN
      # Or
      - REVIEWDOG_GITHUB_API_TOKEN=$$REVIEWDOG_GITHUB_API_TOKEN
    commands:
      - go get github.com/haya14busa/reviewdog/cmd/reviewdog
      - |
        go tool vet -all -shadowstrict . 2>&1 | reviewdog -f=govet -reporter=github-pr-check
      # Or
      - |
        go tool vet -all -shadowstrict . 2>&1 | reviewdog -f=govet -reporter=github-pr-review
    when:
      event: pull_request
```

Finally, run `drone secure` to encrypt the .drone.sec.yaml file and generate a .drone.sec file

```
$ drone secure --repo {github-user-name}/{repo-name} --in .drone.sec.yaml
```

drone.io supports encrypted environment variable for fork Pull Request build in
secure way, but you have to read document carefully http://readme.drone.io/usage/secrets/
not to expose secret data unexpectedly.

#### Common (Jenkins, local, etc...)
You can use reviewdog to post review comments anywhere with following environment variables.

| name | description |
| ---- | ----------- |
| `CI_PULL_REQUEST` | Pull Request number (e.g. 14) |
| `CI_COMMIT`       | SHA1 for the current build |
| `CI_REPO_OWNER`   | repository owner (e.g. "haya14busa" for https://github.com/haya14busa/reviewdog) |
| `CI_REPO_NAME`    | repository name (e.g. "reviewdog" for https://github.com/haya14busa/reviewdog) |
| `CI_BRANCH`       | [optional] branch of the commit |

```sh
$ export CI_PULL_REQUEST=14
$ export CI_REPO_OWNER=haya14busa
$ export CI_REPO_NAME=reviewdog
$ export CI_COMMIT=$(git rev-parse HEAD)
$ REVIEWDOG_TOKEN="<your token>" golint ./... | reviewdog -f=golint -target=github-pr-check
Or
$ REVIEWDOG_GITHUB_API_TOKEN="<your token>" golint ./... | reviewdog -f=golint -target=github-pr-review
```

##### Jenkins with Github pull request builder plugin
- [GitHub pull request builder plugin - Jenkins - Jenkins Wiki](https://wiki.jenkins-ci.org/display/JENKINS/GitHub+pull+request+builder+plugin)

```sh
$ export CI_PULL_REQUEST=${ghprbPullId}
$ export CI_REPO_OWNER=haya14busa
$ export CI_REPO_NAME=reviewdog
$ export CI_COMMIT=${ghprbActualCommit}
$ export REVIEWDOG_INSECURE_SKIP_VERIFY=true # set this as you need
$ REVIEWDOG_TOKEN="<your token>" reviewdog -conf=.reviewdog.yml -reporter=github-pr-check
# Or
$ REVIEWDOG_GITHUB_API_TOKEN="<your token>" reviewdog -conf=.reviewdog.yml -reporter=github-pr-review
```

## :bird: Author
haya14busa (https://github.com/haya14busa)

[travis-badge]: https://travis-ci.org/haya14busa/reviewdog.svg?branch=master
[circleci-badge]: https://circleci.com/gh/haya14busa/reviewdog/tree/master.svg?style=svg

<!-- From https://github.com/zchee/template -->
[godoc-badge]: https://img.shields.io/badge/godoc-reference-4F73B3.svg?style=flat-square&label=%20godoc.org
[codecov-badge]: https://img.shields.io/codecov/c/github/haya14busa/reviewdog.svg?style=flat-square&label=%20%20Codecov%2Eio&logo=data%3Aimage%2Fsvg%2Bxml%3Bcharset%3Dutf-8%3Bbase64%2CPHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSI0MCIgaGVpZ2h0PSI0MCIgdmlld0JveD0iMCAwIDI1NiAyODEiPjxwYXRoIGZpbGw9IiNFRUUiIGQ9Ik0yMTguNTUxIDM3LjQxOUMxOTQuNDE2IDEzLjI4OSAxNjIuMzMgMCAxMjguMDk3IDAgNTcuNTM3LjA0Ny4wOTEgNTcuNTI3LjA0IDEyOC4xMjFMMCAxNDkuODEzbDE2Ljg1OS0xMS40OWMxMS40NjgtNy44MTQgMjQuNzUtMTEuOTQ0IDM4LjQxNy0xMS45NDQgNC4wNzkgMCA4LjE5OC4zNzMgMTIuMjQgMS4xMSAxMi43NDIgMi4zMiAyNC4xNjUgOC4wODkgMzMuNDE0IDE2Ljc1OCAyLjEyLTQuNjcgNC42MTQtOS4yMDkgNy41Ni0xMy41MzZhODguMDgxIDg4LjA4MSAwIDAgMSAzLjgwNS01LjE1Yy0xMS42NTItOS44NC0yNS42NDktMTYuNDYzLTQwLjkyNi0xOS4yNDVhOTAuMzUgOTAuMzUgMCAwIDAtMTYuMTItMS40NTkgODguMzc3IDg4LjM3NyAwIDAgMC0zMi4yOSA2LjA3YzguMzYtNTEuMjIyIDUyLjg1LTg5LjM3IDEwNS4yMy04OS40MDggMjguMzkyIDAgNTUuMDc4IDExLjA1MyA3NS4xNDkgMzEuMTE3IDE2LjAxMSAxNi4wMSAyNi4yNTQgMzYuMDMzIDI5Ljc4OCA1OC4xMTctMTAuMzI5LTQuMDM1LTIxLjIxMi02LjEtMzIuNDAzLTYuMTQ0bC0xLjU2OC0uMDA3YTkwLjk1NyA5MC45NTcgMCAwIDAtMy40MDEuMTExYy0xLjk1NS4xLTMuODk4LjI3Ny01LjgyMS41LS41NzQuMDYzLTEuMTM5LjE1My0xLjcwNy4yMzEtMS4zNzguMTg2LTIuNzUuMzk1LTQuMTA5LjYzOS0uNjAzLjExLTEuMjAzLjIzMS0xLjguMzUxYTkwLjUxNyA5MC41MTcgMCAwIDAtNC4xMTQuOTM3Yy0uNDkyLjEyNi0uOTgzLjI0My0xLjQ3LjM3NGE5MC4xODMgOTAuMTgzIDAgMCAwLTUuMDkgMS41MzhjLS4xLjAzNS0uMjA0LjA2My0uMzA0LjA5NmE4Ny41MzIgODcuNTMyIDAgMCAwLTExLjA1NyA0LjY0OWMtLjA5Ny4wNS0uMTkzLjEwMS0uMjkzLjE1MWE4Ni43IDg2LjcgMCAwIDAtNC45MTIgMi43MDFsLS4zOTguMjM4YTg2LjA5IDg2LjA5IDAgMCAwLTIyLjMwMiAxOS4yNTNjLS4yNjIuMzE4LS41MjQuNjM1LS43ODQuOTU4LTEuMzc2IDEuNzI1LTIuNzE4IDMuNDktMy45NzYgNS4zMzZhOTEuNDEyIDkxLjQxMiAwIDAgMC0zLjY3MiA1LjkxMyA5MC4yMzUgOTAuMjM1IDAgMCAwLTIuNDk2IDQuNjM4Yy0uMDQ0LjA5LS4wODkuMTc1LS4xMzMuMjY1YTg4Ljc4NiA4OC43ODYgMCAwIDAtNC42MzcgMTEuMjcybC0uMDAyLjAwOXYuMDA0YTg4LjAwNiA4OC4wMDYgMCAwIDAtNC41MDkgMjkuMzEzYy4wMDUuMzk3LjAwNS43OTQuMDE5IDEuMTkyLjAyMS43NzcuMDYgMS41NTcuMTA0IDIuMzM4YTk4LjY2IDk4LjY2IDAgMCAwIC4yODkgMy44MzRjLjA3OC44MDQuMTc0IDEuNjA2LjI3NSAyLjQxLjA2My41MTIuMTE5IDEuMDI2LjE5NSAxLjUzNGE5MC4xMSA5MC4xMSAwIDAgMCAuNjU4IDQuMDFjNC4zMzkgMjIuOTM4IDE3LjI2MSA0Mi45MzcgMzYuMzkgNTYuMzE2bDIuNDQ2IDEuNTY0LjAyLS4wNDhhODguNTcyIDg4LjU3MiAwIDAgMCAzNi4yMzIgMTMuNDVsMS43NDYuMjM2IDEyLjk3NC0yMC44MjItNC42NjQtLjEyN2MtMzUuODk4LS45ODUtNjUuMS0zMS4wMDMtNjUuMS02Ni45MTcgMC0zNS4zNDggMjcuNjI0LTY0LjcwMiA2Mi44NzYtNjYuODI5bDIuMjMtLjA4NWMxNC4yOTItLjM2MiAyOC4zNzIgMy44NTkgNDAuMzI1IDExLjk5N2wxNi43ODEgMTEuNDIxLjAzNi0yMS41OGMuMDI3LTM0LjIxOS0xMy4yNzItNjYuMzc5LTM3LjQ0OS05MC41NTQiLz48L3N2Zz4=
