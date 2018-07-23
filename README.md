<p align="center">
  <a href="https://github.com/haya14busa/reviewdog">
    <img alt="reviewdog" src="https://raw.githubusercontent.com/haya14busa/i/d598ed7dc49fefb0018e422e4c43e5ab8f207a6b/reviewdog/reviewdog.logo.png">
  </a>
</p>

<h2 align="center">
  reviewdog - A code review dog who keeps your codebase healthy.
</h2>

<p align="center">
  <a href="https://gitter.im/haya14busa/reviewdog?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge">
    <img alt="Gitter" src="https://img.shields.io/badge/chat-gitter-1DCE73.svg?label=chat&maxAge=43200">
  </a>
  <a href="https://github.com/haya14busa/reviewdog/blob/master/LICENSE">
    <img alt="LICENSE" src="https://img.shields.io/badge/license-MIT-blue.svg?maxAge=43200">
  </a>
  <a href="https://godoc.org/github.com/haya14busa/reviewdog">
    <img alt="GoDoc" src="https://img.shields.io/badge/godoc-reference-4F73B3.svg?label=godoc.org&maxAge=43200">
  </a>
  <a href="https://github.com/haya14busa/reviewdog/releases">
    <img alt="releases" src="https://img.shields.io/github/release/haya14busa/reviewdog.svg?maxAge=43200">
  </a>
  <a href="https://github.com/haya14busa/reviewdog/releases">
    <img alt="Github All Releases" src="https://img.shields.io/github/downloads/haya14busa/reviewdog/total.svg?maxAge=43200">
  </a>
</p>

<p align="center">
  <a href="https://travis-ci.org/haya14busa/reviewdog"><img alt="Travis Status" src="https://img.shields.io/travis/haya14busa/reviewdog/master.svg?label=travis&maxAge=43200"></a>
  <a href="https://circleci.com/gh/haya14busa/reviewdog"><img alt="CircleCI Status" src="https://img.shields.io/circleci/project/github/haya14busa/reviewdog/master.svg?label=circle&maxAge=43200"></a>
  <a href="https://codecov.io/github/haya14busa/reviewdog"><img alt="Coverage Status" src="https://img.shields.io/codecov/c/github/haya14busa/reviewdog/master.svg?maxAge=43200"></a>
  <a href="http://droneio.haya14busa.com/haya14busa/reviewdog"><img alt="drone.io Build Status" src="http://droneio.haya14busa.com/api/badges/haya14busa/reviewdog/status.svg"></a>
</p>

"reviewdog" provides a way to post review comments to code hosting service,
such as GitHub, automatically by integrating with any linter tools with ease.
It uses an output of lint tools and posts them as a comment if findings are in
diff of patches to review.

reviewdog also supports run in the local environment to filter an output of lint tools
by diff.

[design doc](https://docs.google.com/document/d/1mGOX19SSqRowWGbXieBfGPtLnM0BdTkIc9JelTiu6wA/edit?usp=sharing)

## Table of Contents

- [Installation](#installation)
- [Input Format](#input-format)
  * ['errorformat'](#errorformat)
  * [Available pre-defined 'errorformat'](#available-pre-defined-errorformat)
  * [checkstyle format](#checkstyle-format)
- [reviewdog config file](#reviewdog-config-file)
- [Reporters](#reporters)
  * [Reporter: Local (-reporter=local) [default]](#reporter-local--reporterlocal-default)
  * [Reporter: GitHub Checks (-reporter=github-pr-check)](#reporter-github-checks--reportergithub-pr-check)
  * [Reporter: GitHub PullRequest review comment (-reporter=github-pr-review)](#reporter-github-pullrequest-review-comment--reportergithub-pr-review)
  * [Reporter: GitLab MergeRequest discussions (-reporter=gitlab-mr-discussion)](#reporter-gitlab-mergerequest-discussions--reportergitlab-mr-discussion)
  * [Reporter: GitLab MergeRequest commit (-reporter=gitlab-mr-commit)](#reporter-gitlab-mergerequest-commit--reportergitlab-mr-commit)
- [Supported CI services](#supported-ci-services)
  * [Travis CI](#travis-ci)
  * [Circle CI](#circle-ci)
  * [GitLab CI](#gitlab-ci)
  * [Common (Jenkins, local, etc...)](#common-jenkins-local-etc)
    + [Jenkins with Github pull request builder plugin](#jenkins-with-github-pull-request-builder-plugin)
- [Articles](#articles)

[![github-pr-check sample](https://user-images.githubusercontent.com/3797062/40884858-6efd82a0-6756-11e8-9f1a-c6af4f920fb0.png)](https://github.com/haya14busa/reviewdog/pull/131/checks)
![comment in pull-request](https://user-images.githubusercontent.com/3797062/40941822-1d775064-6887-11e8-98e9-4775d37d47f8.png)
![commit status](https://user-images.githubusercontent.com/3797062/40941738-d62acb0a-6886-11e8-858d-7b97aded2a42.png)
[![sample-comment.png](https://raw.githubusercontent.com/haya14busa/i/dc0ccb1e110515ea407c146d99b749018db05c45/reviewdog/sample-comment.png)](https://github.com/haya14busa/reviewdog/pull/24#discussion_r84599728)
![reviewdog-local-demo.gif](https://raw.githubusercontent.com/haya14busa/i/dc0ccb1e110515ea407c146d99b749018db05c45/reviewdog/reviewdog-local-demo.gif)

## Installation

Get [the binary release](https://github.com/haya14busa/reviewdog/releases) (recommended way)

or

```shell
$ go get -u github.com/haya14busa/reviewdog/cmd/reviewdog
```

## Input Format

### 'errorformat'

reviewdog accepts any compiler or linter result from stdin and parses it with
scan-f like [**'errorformat'**](https://github.com/haya14busa/errorformat),
which is the port of Vim's [errorformat](http://vimdoc.sourceforge.net/htmldoc/quickfix.html#errorformat)
feature.

For example, if the result format is `{file}:{line number}:{column number}: {message}`,
errorformat should be `%f:%l:%c: %m` and you can pass it as `-efm` arguments.

```shell
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
if you want to deal with a more complex output. 'errorformat' can handle more
complex output like a multi-line error message.

By this 'errorformat' feature, reviewdog can support any tools output with ease.

### Available pre-defined 'errorformat'

But, you don't have to write 'errorformat' in many cases. reviewdog supports
pre-defined errorformat for major tools.

You can find available errorformat name by `reviewdog -list` and you can use it
with `-f={name}`.

```shell
$ reviewdog -list
golint          linter for Go source code                                       - https://github.com/golang/lint
govet           Vet examines Go source code and reports suspicious problems     - https://golang.org/cmd/vet/
sbt             the interactive build tool                                      - http://www.scala-sbt.org/
...
```

```shell
$ golint ./... | reviewdog -f=golint -diff="git diff master"
```

You can add supported pre-defined 'errorformat' by contributing to [haya14busa/errorformat](https://github.com/haya14busa/errorformat)

### checkstyle format

reviewdog also accepts [checkstyle XML format](http://checkstyle.sourceforge.net/) as well.
If the linter supports checkstyle format as a report format, you can use
-f=checkstyle instead of using 'errorformat'.

```shell
# Local
$ eslint -f checkstyle . | reviewdog -f=checkstyle -diff="git diff"

# CI (overwrite tool name which is shown in review comment by -name arg)
$ eslint -f checkstyle . | reviewdog -f=checkstyle -name="eslint" -reporter=github-pr-check
```

Also, if you want to pass other Json/XML/etc... format to reviewdog, you can write a converter.

```shell
$ <linter> | <convert-to-checkstyle> | reviewdog -f=checkstyle -name="<linter>" -reporter=github-pr-check
```

## reviewdog config file

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

```shell
$ reviewdog -diff="git diff master"
project/run_test.go:61:28: [golint] error strings should not end with punctuation
project/run.go:57:18: [errcheck]        defer os.Setenv(name, os.Getenv(name))
project/run.go:58:12: [errcheck]        os.Setenv(name, "")
# You can use -conf to specify config file path.
$ reviewdog -conf=./.reviewdog.yml -reporter=github-pr-check
```

Output format for project config based run is one of the following formats.

- `<file>: [<tool name>] <message>`
- `<file>:<lnum>: [<tool name>] <message>`
- `<file>:<lnum>:<col>: [<tool name>] <message>`

## Reporters

reviewdog can report results both in local environment and review services as
continuous integration.

### Reporter: Local (-reporter=local) [default]

reviewdog can find newly introduced findings by filtering linter results
using diff. You can pass diff command as `-diff` arg.

```shell
$ golint ./... | reviewdog -f=golint -diff="git diff master"
```

### Reporter: GitHub Checks (-reporter=github-pr-check)

[![github-pr-check sample](https://user-images.githubusercontent.com/3797062/40884858-6efd82a0-6756-11e8-9f1a-c6af4f920fb0.png)](https://github.com/haya14busa/reviewdog/pull/131/checks)

github-pr-check reporter reports results to [GitHub Checks](https://help.github.com/articles/about-status-checks/).
Since GitHub Checks API is only for GitHub Apps, reviewdog CLI send a request to
reviewdog GitHub App server and the server post results as GitHub Checks.

1. Install reviewdog Apps. https://github.com/apps/reviewdog
2. Set `REVIEWDOG_TOKEN` or run reviewdog CLI in trusted CI providers.
  - Get token from `https://reviewdog.app/gh/{owner}/{repo-name}`.

```shell
$ export REVIEWDOG_TOKEN="<token>"
$ reviewdog -reporter=github-pr-check
```

Note: Token is not required if you run reviewdog in Travis or AppVeyor.

#### *Caution*

As described above, github-pr-check reporter is depending on reviewdog GitHub
App server.
The server is running with haya14busa's pocket money for now and I may break
things, so I cannot ensure that the server is running 24h and 365 days.

**UPDATE:** Started getting support by [opencollective](https://opencollective.com/reviewdog).
See [Supporting reviewdog](#supporting-reviewdog)

github-pr-check reporter is better than github-pr-review reporter in general
because it provides more rich feature and has less scope, but please bear in
mind the above caution and please use it on your own risk.

You can use github-pr-review reporter if you don't want to depend on reviewdog
server.

### Reporter: GitHub PullRequest review comment (-reporter=github-pr-review)

[![sample-comment.png](https://raw.githubusercontent.com/haya14busa/i/dc0ccb1e110515ea407c146d99b749018db05c45/reviewdog/sample-comment.png)](https://github.com/haya14busa/reviewdog/pull/24#discussion_r84599728)

github-pr-review reporter reports results to GitHub PullRequest review comments
using GitHub Personal API Access Token.
[GitHub Enterprise](https://enterprise.github.com/home) is supported too.

- Go to https://github.com/settings/tokens and generate new API token.
- Check `repo` for private repositories or `public_repo` for public repositories.

```shell
$ export REVIEWDOG_GITHUB_API_TOKEN="<token>"
$ reviewdog -reporter=github-pr-review`
```

For GitHub Enterprise, set API endpoint by environment variable.

```shell
$ export GITHUB_API="https://example.githubenterprise.com/api/v3/"
$ export REVIEWDOG_INSECURE_SKIP_VERIFY=true # set this as you need to skip verifying SSL
```

### Reporter: GitLab MergeRequest discussions (-reporter=gitlab-mr-discussion)

[![gitlab-mr-discussion sample](https://user-images.githubusercontent.com/3797062/41810718-f91bc540-773d-11e8-8598-fbc09ce9b1c7.png)](https://gitlab.com/haya14busa/reviewdog/merge_requests/113#note_83411103)

Required GitLab version: >= v10.8.0

gitlab-mr-discussion reporter reports results to GitLab MergeRequest discussions using
GitLab Personal API Access token.
Get the token with `api` scope from https://gitlab.com/profile/personal_access_tokens.

```shell
$ export REVIEWDOG_GITLAB_API_TOKEN="<token>"
$ reviewdog -reporter=gitlab-mr-discussion
```

For self-hosted GitLab, set API endpoint by environment variable.

```shell
$ export GITLAB_API="https://example.gitlab.com/api/v4"
$ export REVIEWDOG_INSECURE_SKIP_VERIFY=true # set this as you need to skip verifying SSL
```

### Reporter: GitLab MergeRequest commit (-reporter=gitlab-mr-commit)

gitlab-mr-commit is similar to [gitlab-mr-discussion](#reporter-gitlab-mergerequest-discussions--reportergitlab-mr-discussion) reporter but reports results to each commit in GitLab MergeRequest.

gitlab-mr-discussion is recommended, but you can use gitlab-mr-commit reporter
if your GitLab version is under v10.8.0.

```shell
$ export REVIEWDOG_GITLAB_API_TOKEN="<token>"
$ reviewdog -reporter=gitlab-mr-commit
```

## Supported CI services

### Travis CI

#### Travis CI (-reporter=github-pr-check)

If you use -reporter=github-pr-check in Travis CI, you don't need to set `REVIEWDOG_TOKEN`.

Example:

```yaml
env:
  global:
    - REVIEWDOG_VERSION="0.9.11"

install:
  - mkdir -p ~/bin/ && export export PATH="~/bin/:$PATH"
  - curl -fSL https://github.com/haya14busa/reviewdog/releases/download/$REVIEWDOG_VERSION/reviewdog_linux_amd64 -o ~/bin/reviewdog && chmod +x ~/bin/reviewdog

script:
  - reviewdog -conf=.reviewdog.yml -reporter=github-pr-check
```

#### Travis CI (-reporter=github-pr-review)

Store GitHub API token by [travis encryption keys](https://docs.travis-ci.com/user/encryption-keys/).

```shell
$ gem install travis
$ travis encrypt REVIEWDOG_GITHUB_API_TOKEN=<token> --add env.global
```
Example:

```yaml
env:
  global:
    - secure: <token>
    - REVIEWDOG_VERSION="0.9.11"

install:
  - mkdir -p ~/bin/ && export export PATH="~/bin/:$PATH"
  - curl -fSL https://github.com/haya14busa/reviewdog/releases/download/$REVIEWDOG_VERSION/reviewdog_linux_amd64 -o ~/bin/reviewdog && chmod +x ~/bin/reviewdog

script:
  - >-
    golint ./... | reviewdog -f=golint -reporter=github-pr-review
```

Examples
- https://github.com/azu/textlint-reviewdog-example

### Circle CI

Store `REVIEWDOG_TOKEN` or `REVIEWDOG_GITHUB_API_TOKEN` in
[Environment variables - CircleCI](https://circleci.com/docs/environment-variables/#setting-environment-variables-for-all-commands-without-adding-them-to-git)

#### .circleci/config.yml sample

```yaml
version: 2
jobs:
  build:
    docker:
      - image: golang:latest
        environment:
          REVIEWDOG_VERSION: "0.9.11"
    steps:
      - checkout
      - run: curl -fSL https://github.com/haya14busa/reviewdog/releases/download/$REVIEWDOG_VERSION/reviewdog_linux_amd64 -o reviewdog && chmod +x ./reviewdog
      - run: go vet ./... 2>&1 | ./reviewdog -f=govet -reporter=github-pr-check
      # or
      - run: go vet ./... 2>&1 | ./reviewdog -f=govet -reporter=github-pr-review
```

### GitLab CI

Store `REVIEWDOG_GITLAB_API_TOKEN` in [GitLab CI variable](https://docs.gitlab.com/ee/ci/variables/#variables).

#### .gitlab-ci.yml sample

```yaml
reviewdog:
  script:
    - reviewdog -reporter=gitlab-mr-discussion
    # Or
    - reviewdog -reporter=gitlab-mr-commit
```

### Common (Jenkins, local, etc...)

You can use reviewdog to post review comments from anywhere with following
environment variables.

| name | description |
| ---- | ----------- |
| `CI_PULL_REQUEST` | Pull Request number (e.g. 14) |
| `CI_COMMIT`       | SHA1 for the current build |
| `CI_REPO_OWNER`   | repository owner (e.g. "haya14busa" for https://github.com/haya14busa/reviewdog) |
| `CI_REPO_NAME`    | repository name (e.g. "reviewdog" for https://github.com/haya14busa/reviewdog) |
| `CI_BRANCH`       | [optional] branch of the commit |

```shell
$ export CI_PULL_REQUEST=14
$ export CI_REPO_OWNER=haya14busa
$ export CI_REPO_NAME=reviewdog
$ export CI_COMMIT=$(git rev-parse HEAD)
```
and set a token if required.

```shell
$ REVIEWDOG_TOKEN="<token>"
$ REVIEWDOG_GITHUB_API_TOKEN="<token>"
$ REVIEWDOG_GITLAB_API_TOKEN="<token>"
```

#### Jenkins with Github pull request builder plugin
- [GitHub pull request builder plugin - Jenkins - Jenkins Wiki](https://wiki.jenkins-ci.org/display/JENKINS/GitHub+pull+request+builder+plugin)

```shell
$ export CI_PULL_REQUEST=${ghprbPullId}
$ export CI_REPO_OWNER=haya14busa
$ export CI_REPO_NAME=reviewdog
$ export CI_COMMIT=${ghprbActualCommit}
$ export REVIEWDOG_INSECURE_SKIP_VERIFY=true # set this as you need
$ REVIEWDOG_TOKEN="<token>" reviewdog -reporter=github-pr-check
# Or
$ REVIEWDOG_GITHUB_API_TOKEN="<token>" reviewdog -reporter=github-pr-review
```

## Articles
- [reviewdog — A code review dog who keeps your codebase healthy ](https://medium.com/@haya14busa/reviewdog-a-code-review-dog-who-keeps-your-codebase-healthy-d957c471938b)
- [reviewdog ♡ GitHub Check — improved automated review experience](https://medium.com/@haya14busa/reviewdog-github-check-improved-automated-review-experience-58f89e0c95f3)

## :bird: Author
haya14busa [![GitHub followers](https://img.shields.io/github/followers/haya14busa.svg?style=social&label=Follow)](https://github.com/haya14busa)

## Contributors

[![Contributors](https://opencollective.com/reviewdog/contributors.svg?width=890)](https://github.com/haya14busa/reviewdog/graphs/contributors)

### Supporting reviewdog
Become a backer or sponsor and get your logo on our README on Github with a link to your site.

[![Become a backer](https://opencollective.com/reviewdog/tiers/backer.svg?avatarHeight=64)](https://opencollective.com/reviewdog#backers)
[![Become a sponsor](https://opencollective.com/reviewdog/tiers/sponsor.svg?avatarHeight=64)](https://opencollective.com/reviewdog#sponsor)
