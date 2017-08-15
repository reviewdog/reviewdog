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


Automatic code review ([sample PR](https://github.com/haya14busa/reviewdog/pull/24#discussion_r84599728))

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
$ eslint -f checkstyle . | reviewdog -f=checkstyle -name="eslint" -ci="circle-ci"
```

Also, if you want to pass other Json/XML/etc... format to reviewdog, you can write a converter.

```
$ <linter> | <convert-to-checkstyle> | reviewdog -f=checkstyle -name="<linter>" -ci="circle-ci"
```

### Project Configuration Based Run

[experimental]

reviewdog can also be controlled via the reviewdog.yml configuration file instead of "-f" or "-efm" arguments.

With reviewdog.yml, you can run the same commands both CI service and local
environment including editor intergration with ease.

#### reviewdog.yml

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
$ reviewdog -ci=droneio -conf=./reviewdog.yml
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

reviewdog requires GitHub Personal API Access token as an environment variable
(`REVIEWDOG_GITHUB_API_TOKEN`) to post comments to GitHub or GitHub Enterprise.
Go to https://github.com/settings/tokens and generate new API token.
Check `repo` for private repositories or `public_repo` for public repositories.
Export the token environment variable by secure way, depending on CI services.

For GitHub Enterprise, please set API endpoint by environment variable.

```
export GITHUB_API="https://example.githubenterprise.com/api/v3"
```

#### Supported CI services

| Name | Pull Request from the same repository | Pull Request from forked repository |
| ---- | ------------------------------------- | ----------------------------------- |
| [Travis CI](https://travis-ci.org/) | :o: | :x:
| [CircleCI](https://circleci.com/) | :o: | :x: (but possible with insecure way)
| [drone.io](https://github.com/drone/drone) (OSS) v0.4 | :o: | :o:
| common (Your managed CI server like Jenkins) | :o: | :o:

reviewdog can run in CI services which supports Pull Request build and secret
environment variable feature for security reason.

But, for the Pull Request from forked repository, most CI services restrict
secret environment variable for security reason, to avoid leak of secret data
with malicious Pull Request. ([Travis CI](https://docs.travis-ci.com/user/pull-requests#Pull-Requests-and-Security-Restrictions), [CircleCI](https://circleci.com/docs/fork-pr-builds/#security-implications-of-running-builds-for-pull-requests-from-forks))


##### Travis CI

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
    - REVIEWDOG_VERSION=0.9.8

install:
  - mkdir -p ~/bin/ && export export PATH="~/bin/:$PATH"
  - curl -fSL https://github.com/haya14busa/reviewdog/releases/download/$REVIEWDOG_VERSION/reviewdog_linux_amd64 -o ~/bin/reviewdog && chmod +x ~/bin/reviewdog

script:
  - >-
    golint ./... | reviewdog -f=golint -ci=travis
```

Examples
- https://github.com/azu/textlint-reviewdog-example

##### Circle CI

Store GitHub API token in [Environment variables - CircleCI](https://circleci.com/docs/environment-variables/#setting-environment-variables-for-all-commands-without-adding-them-to-git)

Circle CI do not build by pull-request hook by default, so please turn on "Only
build pull requests" in Advanced option in Circle CI
([changelog](https://circleci.com/changelog/#only-pr-builds)).

Pull Requests from fork repo cannot set environment variable for security reason.
However, if you want to run fork PR builds for private repositories or want to run
reviewdog for forked PR to OSS project, CircleCI have an option to enable it.
[Unsafe fork PR builds](https://circleci.com/docs/fork-pr-builds/#unsafe-fork-pr-builds)

I thinks it's not a big problem if GitHub API token has limited scope (e.g. only `public_repo`),
but if you enables [Unsafe fork PR builds](https://circleci.com/docs/fork-pr-builds/#unsafe-fork-pr-builds)
to run reviewdog for fork PR, please use it at your own risk.

circle.yml sample

```yaml
machine:
  environment:
    REVIEWDOG_VERSION: 0.9.8

dependencies:
  override:
    - curl -fSL https://github.com/haya14busa/reviewdog/releases/download/$REVIEWDOG_VERSION/reviewdog_linux_amd64 -o reviewdog && chmod +x ./reviewdog

test:
  override:
    - >-
      go tool vet -all -shadowstrict . 2>&1 | ./reviewdog -f=govet -ci="circle-ci"
```

##### drone.io
Store GitHub API token in environment variable.  [Secrets · Drone](http://readme.drone.io/usage/secrets/)

Install 'drone' cli command http://readme.drone.io/devs/cli/ and setup configuration.

```
echo '.drone.sec.yaml' >> .gitignore
```

.drone.sec.yaml

```yaml
environment:
  REVIEWDOG_GITHUB_API_TOKEN: <your token>
```

.drone.yaml example

```yaml
build:
  lint:
    image: golang
    environment:
      - REVIEWDOG_GITHUB_API_TOKEN=$$REVIEWDOG_GITHUB_API_TOKEN
    commands:
      - go get github.com/haya14busa/reviewdog/cmd/reviewdog
      - |
        go tool vet -all -shadowstrict . 2>&1 | reviewdog -f=govet -ci=droneio
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

##### Common (Jenkins, local, etc...)
You can use reviewdog to post review comments anywhere with following environment variables.

| name | description |
| ---- | ----------- |
| `CI_PULL_REQUEST` | Pull Request number (e.g. 14) |
| `CI_COMMIT`       | SHA1 for the current build |
| `CI_REPO_OWNER`   | repository owner (e.g. "haya14busa" for https://github.com/haya14busa/reviewdog) |
| `CI_REPO_NAME`    | repository name (e.g. "reviewdog" for https://github.com/haya14busa/reviewdog) |
| `REVIEWDOG_GITHUB_API_TOKEN`    | GitHub Personal API Access token |

```sh
$ export CI_PULL_REQUEST=14
$ export CI_REPO_OWNER=haya14busa
$ export CI_REPO_NAME=reviewdog
$ export CI_COMMIT=$(git rev-parse HEAD)
$ export REVIEWDOG_GITHUB_API_TOKEN="<your token>"
$ golint ./... | reviewdog -f=golint -ci=common
```

##### Jenkins with Github pull request builder plugin
- [GitHub pull request builder plugin - Jenkins - Jenkins Wiki](https://wiki.jenkins-ci.org/display/JENKINS/GitHub+pull+request+builder+plugin)

```sh
$ export CI_PULL_REQUEST=${ghprbPullId}
$ export CI_REPO_OWNER=haya14busa
$ export CI_REPO_NAME=reviewdog
$ export CI_COMMIT=${ghprbActualCommit}
$ export REVIEWDOG_GITHUB_API_TOKEN="<your token>"
$ export REVIEWDOG_INSECURE_SKIP_VERIFY=true # set this as you need
$ reviewdog -ci=common -conf=reviewdog.yml
```

## :bird: Author
haya14busa (https://github.com/haya14busa)

<!-- From https://github.com/zchee/template -->
[travis-badge]: https://img.shields.io/travis/haya14busa/reviewdog.svg?style=flat-square&label=%20Travis%20CI&logo=data%3Aimage%2Fsvg%2Bxml%3Bcharset%3Dutf-8%3Bbase64%2CPHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSI2MCIgaGVpZ2h0PSI2MCIgdmlld0JveD0iNSA0IDI0IDI0Ij48cGF0aCBmaWxsPSIjREREIiBkPSJNMTEuMzkyKzkuMzc0aDQuMDk2djEzLjEyaC0xLjUzNnYyLjI0aDYuMDgwdi0yLjQ5NmgtMS45MnYtMTMuMDU2aDQuMzUydjEuOTJoMS45ODR2LTMuOTA0aC0xNS4yOTZ2My45MDRoMi4yNHpNMjkuMjYzKzIuNzE4aC0yNC44NDhjLTAuNDMzKzAtMC44MzIrMC4zMjEtMC44MzIrMC43NDl2MjQuODQ1YzArMC40MjgrMC4zOTgrMC43NzQrMC44MzIrMC43NzRoMjQuODQ4YzAuNDMzKzArMC43NTMtMC4zNDcrMC43NTMtMC43NzR2LTI0Ljg0NWMwLTAuNDI4LTAuMzE5LTAuNzQ5LTAuNzUzLTAuNzQ5ek0yNS43MjgrMTIuMzgyaC00LjU0NHYtMS45MmgtMS43OTJ2MTAuNDk2aDEuOTJ2NS4wNTZoLTguNjR2LTQuOGgxLjUzNnYtMTAuNTZoLTEuNTM2djEuNzI4aC00Ljh2LTYuNDY0aDE3Ljg1NnY2LjQ2NHoiLz48L3N2Zz4=
[circleci-badge]: https://img.shields.io/circleci/project/github/haya14busa/reviewdog.svg?style=flat-square&label=%20%20CircleCI&logoWidth=16&logo=data%3Aimage%2Fsvg%2Bxml%3Bcharset%3Dutf-8%3Bbase64%2CPHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSI0MCIgdmlld0JveD0iMCAwIDIwMCAyMDAiPjxwYXRoIGZpbGw9IiNEREQiIGQ9Ik03NC43IDEwMGMwLTEzLjIgMTAuNy0yMy44IDIzLjgtMjMuOCAxMy4xIDAgMjMuOCAxMC43IDIzLjggMjMuOCAwIDEzLjEtMTAuNyAyMy44LTIzLjggMjMuOC0xMy4xIDAtMjMuOC0xMC43LTIzLjgtMjMuOHpNOTguNSAwQzUxLjggMCAxMi43IDMyIDEuNiA3NS4yYy0uMS4zLS4xLjYtLjEgMSAwIDIuNiAyLjEgNC44IDQuOCA0LjhoNDAuM2MxLjkgMCAzLjYtMS4xIDQuMy0yLjggOC4zLTE4IDI2LjUtMzAuNiA0Ny42LTMwLjYgMjguOSAwIDUyLjQgMjMuNSA1Mi40IDUyLjRzLTIzLjUgNTIuNC01Mi40IDUyLjRjLTIxLjEgMC0zOS4zLTEyLjUtNDcuNi0zMC42LS44LTEuNi0yLjQtMi44LTQuMy0yLjhINi4zYy0yLjYgMC00LjggMi4xLTQuOCA0LjggMCAuMy4xLjYuMSAxQzEyLjYgMTY4IDUxLjggMjAwIDk4LjUgMjAwYzU1LjIgMCAxMDAtNDQuOCAxMDAtMTAwUzE1My43IDAgOTguNSAweiIvPjwvc3ZnPg%3D%3D
[godoc-badge]: https://img.shields.io/badge/godoc-reference-4F73B3.svg?style=flat-square&label=%20godoc.org
[codecov-badge]: https://img.shields.io/codecov/c/github/haya14busa/reviewdog.svg?style=flat-square&label=%20%20Codecov%2Eio&logo=data%3Aimage%2Fsvg%2Bxml%3Bcharset%3Dutf-8%3Bbase64%2CPHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSI0MCIgaGVpZ2h0PSI0MCIgdmlld0JveD0iMCAwIDI1NiAyODEiPjxwYXRoIGZpbGw9IiNFRUUiIGQ9Ik0yMTguNTUxIDM3LjQxOUMxOTQuNDE2IDEzLjI4OSAxNjIuMzMgMCAxMjguMDk3IDAgNTcuNTM3LjA0Ny4wOTEgNTcuNTI3LjA0IDEyOC4xMjFMMCAxNDkuODEzbDE2Ljg1OS0xMS40OWMxMS40NjgtNy44MTQgMjQuNzUtMTEuOTQ0IDM4LjQxNy0xMS45NDQgNC4wNzkgMCA4LjE5OC4zNzMgMTIuMjQgMS4xMSAxMi43NDIgMi4zMiAyNC4xNjUgOC4wODkgMzMuNDE0IDE2Ljc1OCAyLjEyLTQuNjcgNC42MTQtOS4yMDkgNy41Ni0xMy41MzZhODguMDgxIDg4LjA4MSAwIDAgMSAzLjgwNS01LjE1Yy0xMS42NTItOS44NC0yNS42NDktMTYuNDYzLTQwLjkyNi0xOS4yNDVhOTAuMzUgOTAuMzUgMCAwIDAtMTYuMTItMS40NTkgODguMzc3IDg4LjM3NyAwIDAgMC0zMi4yOSA2LjA3YzguMzYtNTEuMjIyIDUyLjg1LTg5LjM3IDEwNS4yMy04OS40MDggMjguMzkyIDAgNTUuMDc4IDExLjA1MyA3NS4xNDkgMzEuMTE3IDE2LjAxMSAxNi4wMSAyNi4yNTQgMzYuMDMzIDI5Ljc4OCA1OC4xMTctMTAuMzI5LTQuMDM1LTIxLjIxMi02LjEtMzIuNDAzLTYuMTQ0bC0xLjU2OC0uMDA3YTkwLjk1NyA5MC45NTcgMCAwIDAtMy40MDEuMTExYy0xLjk1NS4xLTMuODk4LjI3Ny01LjgyMS41LS41NzQuMDYzLTEuMTM5LjE1My0xLjcwNy4yMzEtMS4zNzguMTg2LTIuNzUuMzk1LTQuMTA5LjYzOS0uNjAzLjExLTEuMjAzLjIzMS0xLjguMzUxYTkwLjUxNyA5MC41MTcgMCAwIDAtNC4xMTQuOTM3Yy0uNDkyLjEyNi0uOTgzLjI0My0xLjQ3LjM3NGE5MC4xODMgOTAuMTgzIDAgMCAwLTUuMDkgMS41MzhjLS4xLjAzNS0uMjA0LjA2My0uMzA0LjA5NmE4Ny41MzIgODcuNTMyIDAgMCAwLTExLjA1NyA0LjY0OWMtLjA5Ny4wNS0uMTkzLjEwMS0uMjkzLjE1MWE4Ni43IDg2LjcgMCAwIDAtNC45MTIgMi43MDFsLS4zOTguMjM4YTg2LjA5IDg2LjA5IDAgMCAwLTIyLjMwMiAxOS4yNTNjLS4yNjIuMzE4LS41MjQuNjM1LS43ODQuOTU4LTEuMzc2IDEuNzI1LTIuNzE4IDMuNDktMy45NzYgNS4zMzZhOTEuNDEyIDkxLjQxMiAwIDAgMC0zLjY3MiA1LjkxMyA5MC4yMzUgOTAuMjM1IDAgMCAwLTIuNDk2IDQuNjM4Yy0uMDQ0LjA5LS4wODkuMTc1LS4xMzMuMjY1YTg4Ljc4NiA4OC43ODYgMCAwIDAtNC42MzcgMTEuMjcybC0uMDAyLjAwOXYuMDA0YTg4LjAwNiA4OC4wMDYgMCAwIDAtNC41MDkgMjkuMzEzYy4wMDUuMzk3LjAwNS43OTQuMDE5IDEuMTkyLjAyMS43NzcuMDYgMS41NTcuMTA0IDIuMzM4YTk4LjY2IDk4LjY2IDAgMCAwIC4yODkgMy44MzRjLjA3OC44MDQuMTc0IDEuNjA2LjI3NSAyLjQxLjA2My41MTIuMTE5IDEuMDI2LjE5NSAxLjUzNGE5MC4xMSA5MC4xMSAwIDAgMCAuNjU4IDQuMDFjNC4zMzkgMjIuOTM4IDE3LjI2MSA0Mi45MzcgMzYuMzkgNTYuMzE2bDIuNDQ2IDEuNTY0LjAyLS4wNDhhODguNTcyIDg4LjU3MiAwIDAgMCAzNi4yMzIgMTMuNDVsMS43NDYuMjM2IDEyLjk3NC0yMC44MjItNC42NjQtLjEyN2MtMzUuODk4LS45ODUtNjUuMS0zMS4wMDMtNjUuMS02Ni45MTcgMC0zNS4zNDggMjcuNjI0LTY0LjcwMiA2Mi44NzYtNjYuODI5bDIuMjMtLjA4NWMxNC4yOTItLjM2MiAyOC4zNzIgMy44NTkgNDAuMzI1IDExLjk5N2wxNi43ODEgMTEuNDIxLjAzNi0yMS41OGMuMDI3LTM0LjIxOS0xMy4yNzItNjYuMzc5LTM3LjQ0OS05MC41NTQiLz48L3N2Zz4=
