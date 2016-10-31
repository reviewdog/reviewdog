## reviewdog - A code review dog who keeps your codebase healthy

[![Gitter](https://badges.gitter.im/haya14busa/reviewdog.svg)](https://gitter.im/haya14busa/reviewdog?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)
[![LICENSE](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/haya14busa/reviewdog)](https://goreportcard.com/report/github.com/haya14busa/reviewdog)
[![releases](https://img.shields.io/github/release/haya14busa/reviewdog.svg)](https://github.com/haya14busa/reviewdog/releases)

![reviewdog logo](https://raw.githubusercontent.com/haya14busa/i/d598ed7dc49fefb0018e422e4c43e5ab8f207a6b/reviewdog/reviewdog.logo.png)

| CI | Status |
| ---- | ----------- |
| [Travis CI](https://travis-ci.org/haya14busa/reviewdog) | [![Travis Build Status](https://travis-ci.org/haya14busa/reviewdog.svg?branch=master)](https://travis-ci.org/haya14busa/reviewdog) |
| [CircleCI](https://circleci.com/gh/haya14busa/reviewdog) | [![CircleCI](https://circleci.com/gh/haya14busa/reviewdog.svg?style=svg)](https://circleci.com/gh/haya14busa/reviewdog) |
| [drone.io](http://droneio.haya14busa.com/haya14busa/reviewdog) | [![drone.io Build Status](http://droneio.haya14busa.com/api/badges/haya14busa/reviewdog/status.svg)](http://droneio.haya14busa.com/haya14busa/reviewdog) |

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

### Run locally

reviewdog can find new introduced warnings or error by filtering linter results
using diff. You can pass diff comamnd as `-diff` arg, like `-diff="git diff"`,
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
    secure: xxxxxxxxxxxxx

install:
  - go get github.com/haya14busa/reviewdog/cmd/reviewdog

script:
  - >-
    golint ./... | reviewdog -f=golint -ci=travis
```

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
    REVIEWDOG_VERSION: 0.9.1

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

## :bird: Author
haya14busa (https://github.com/haya14busa)
