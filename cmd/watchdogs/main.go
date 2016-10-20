package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
	"github.com/haya14busa/errorformat"
	"github.com/haya14busa/watchdogs"
	"github.com/mattn/go-shellwords"
)

const usageMessage = "" +
	`Usage: watchdogs [flags]
`

// flags
var (
	diffCmd   string
	diffStrip int
	efms      strslice
	ci        string
)

func init() {
	flag.StringVar(&diffCmd, "diff", "", "diff command for filitering checker results")
	flag.IntVar(&diffStrip, "strip", 1, "strip NUM leading components from diff file names (equivalent to `patch -p`) (default is 1 for git diff)")
	flag.Var(&efms, "efm", "list of errorformat")
	flag.StringVar(&ci, "ci", "", "CI service (supported travis, circle-ci, droneio(OSS 0.4))")
}

func usage() {
	fmt.Fprintln(os.Stderr, usageMessage)
	fmt.Fprintln(os.Stderr, "Flags:")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if err := run(os.Stdin, os.Stdout, diffCmd, diffStrip, efms, ci); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(r io.Reader, w io.Writer, diffCmd string, diffStrip int, efms []string, ci string) error {
	p, err := efmParser(efms)
	if err != nil {
		return err
	}

	var cs watchdogs.CommentService
	var ds watchdogs.DiffService

	if ci != "" {
		gs, isPR, err := githubService(ci)
		if err != nil {
			return err
		}
		if !isPR {
			fmt.Fprintf(os.Stderr, "this is not PullRequest build. CI: %v\n", ci)
			return nil
		}
		cs = gs
		ds = gs
	} else {
		// local
		cs = watchdogs.NewCommentWriter(w)
		d, err := diffService(diffCmd, diffStrip)
		if err != nil {
			return err
		}
		ds = d
	}

	app := watchdogs.NewWatchdogs(p, cs, ds)
	if err := app.Run(r); err != nil {
		return err
	}
	if fcs, ok := cs.(FlashCommentService); ok {
		// Output log to writer
		for _, c := range fcs.ListPostComments() {
			fmt.Fprintln(w, strings.Join(c.Lines, "\n"))
		}
		return fcs.Flash()
	}
	return nil
}

// FlashCommentService is CommentService which uses Flash method to post comment.
type FlashCommentService interface {
	watchdogs.CommentService
	ListPostComments() []*watchdogs.Comment
	Flash() error
}

func efmParser(efms []string) (watchdogs.Parser, error) {
	efm, err := errorformat.NewErrorformat(efms)
	if err != nil {
		return nil, err
	}
	return watchdogs.NewErrorformatParser(efm), nil
}

func diffService(s string, strip int) (watchdogs.DiffService, error) {
	cmds, err := shellwords.Parse(s)
	if err != nil {
		return nil, err
	}
	if len(cmds) < 1 {
		return nil, errors.New("diff command is empty")
	}
	cmd := exec.Command(cmds[0], cmds[1:]...)
	d := watchdogs.NewDiffCmd(cmd, strip)
	return d, nil
}

func githubService(ci string) (githubservice *watchdogs.GitHubPullRequest, isPR bool, err error) {
	token, err := nonEmptyEnv("WATCHDOGS_GITHUB_API_TOKEN")
	if err != nil {
		return nil, false, err
	}
	var g *GitHubPR
	switch ci {
	case "travis":
		g, isPR, err = travis()
	case "circle-ci":
		g, isPR, err = circleci()
	case "droneio":
		g, isPR, err = droneio()
	default:
		return nil, false, fmt.Errorf("unsupported CI: %v", ci)
	}
	if err != nil {
		return nil, false, err
	}
	// TODO: support commit build
	if !isPR {
		return nil, false, nil
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	client := github.NewClient(tc)
	githubservice = watchdogs.NewGitHubPullReqest(client, g.owner, g.repo, g.pr, g.sha)
	return githubservice, true, nil
}

func travis() (g *GitHubPR, isPR bool, err error) {
	prs := os.Getenv("TRAVIS_PULL_REQUEST")
	if prs == "false" {
		return nil, false, nil
	}
	pr, err := strconv.Atoi(prs)
	if err != nil {
		return nil, true, fmt.Errorf("unexpected env variable. TRAVIS_PULL_REQUEST=%v", prs)
	}
	reposlug, err := nonEmptyEnv("TRAVIS_REPO_SLUG")
	if err != nil {
		return nil, true, err
	}
	rss := strings.SplitN(reposlug, "/", 2)
	if len(rss) < 2 {
		return nil, true, fmt.Errorf("unexpected env variable. TRAVIS_REPO_SLUG=%v", reposlug)
	}
	owner, repo := rss[0], rss[1]

	sha, err := nonEmptyEnv("TRAVIS_PULL_REQUEST_SHA")
	if err != nil {
		return nil, true, err
	}

	g = &GitHubPR{
		owner: owner,
		repo:  repo,
		pr:    pr,
		sha:   sha,
	}
	return g, true, nil
}

// https://circleci.com/docs/environment-variables/
func circleci() (g *GitHubPR, isPR bool, err error) {
	var prs string // pull request number in string
	// For Pull Request from a same repository
	// e.g. https: //github.com/haya14busa/watchdogs/pull/6
	// it might be better to support CI_PULL_REQUESTS instead.
	prs = os.Getenv("CI_PULL_REQUEST")
	if prs == "" {
		// For Pull Request by a fork repository
		// e.g. 6
		prs = os.Getenv("CIRCLE_PR_NUMBER")
	}
	if prs == "" {
		// not a pull-request build
		return nil, false, nil
	}
	// regexp.MustCompile() in func intentionally because this func is called
	// once for one run.
	re := regexp.MustCompile(`[1-9]\d*$`)
	prm := re.FindString(prs)
	pr, err := strconv.Atoi(prm)
	if err != nil {
		return nil, true, fmt.Errorf("unexpected env variable (CI_PULL_REQUEST or CIRCLE_PR_NUMBER): %v", prs)
	}
	owner, err := nonEmptyEnv("CIRCLE_PROJECT_USERNAME")
	if err != nil {
		return nil, true, err
	}
	repo, err := nonEmptyEnv("CIRCLE_PROJECT_REPONAME")
	if err != nil {
		return nil, true, err
	}
	sha, err := nonEmptyEnv("CIRCLE_SHA1")
	if err != nil {
		return nil, true, err
	}
	g = &GitHubPR{
		owner: owner,
		repo:  repo,
		pr:    pr,
		sha:   sha,
	}
	return g, true, nil
}

// http://readme.drone.io/usage/variables/
func droneio() (g *GitHubPR, isPR bool, err error) {
	var prs string // pull request number in string
	prs = os.Getenv("DRONE_PULL_REQUEST")
	if prs == "" {
		// not a pull-request build
		return nil, false, nil
	}
	pr, err := strconv.Atoi(prs)
	if err != nil {
		return nil, true, fmt.Errorf("unexpected env variable (DRONE_PULL_REQUEST): %v", prs)
	}
	reposlug, err := nonEmptyEnv("DRONE_REPO") // e.g. haya14busa/watchdogs
	if err != nil {
		return nil, true, err
	}
	rss := strings.SplitN(reposlug, "/", 2)
	if len(rss) < 2 {
		return nil, true, fmt.Errorf("unexpected env variable. DRONE_REPO=%v", reposlug)
	}
	owner, repo := rss[0], rss[1]
	sha, err := nonEmptyEnv("DRONE_COMMIT")
	if err != nil {
		return nil, true, err
	}
	g = &GitHubPR{
		owner: owner,
		repo:  repo,
		pr:    pr,
		sha:   sha,
	}
	return g, true, nil
}

// GitHubPR represents required information about GitHub PullRequest.
type GitHubPR struct {
	owner string
	repo  string
	pr    int
	sha   string
}

func nonEmptyEnv(env string) (string, error) {
	v := os.Getenv(env)
	if v == "" {
		return "", fmt.Errorf("environment variable $%v is not set", env)
	}
	return v, nil
}

type strslice []string

func (ss *strslice) String() string {
	return fmt.Sprintf("%v", *ss)
}

func (ss *strslice) Set(value string) error {
	*ss = append(*ss, value)
	return nil
}
