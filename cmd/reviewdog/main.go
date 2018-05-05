package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"golang.org/x/net/context" // "context"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
	"github.com/haya14busa/errorformat/fmts"
	"github.com/haya14busa/reviewdog"
	"github.com/haya14busa/reviewdog/project"
	shellwords "github.com/mattn/go-shellwords"
	"github.com/xanzy/go-gitlab"
)

const version = "0.9.8"

const usageMessage = "" +
	`Usage:	reviewdog [flags]
	reviewdog accepts any compiler or linter results from stdin and filters
	them by diff for review. reviewdog also can posts the results as a comment to
	GitHub if you use reviewdog in CI service.
`

type option struct {
	version   bool
	diffCmd   string
	diffStrip int
	efms      strslice
	f         string // errorformat name
	list      bool   // list supported errorformat name
	name      string // tool name which is used in comment
	ci        string
	conf      string
}

// flags doc
const (
	diffCmdDoc   = `diff command (e.g. "git diff"). diff flag is ignored if you pass "ci" flag`
	diffStripDoc = "strip NUM leading components from diff file names (equivalent to 'patch -p') (default is 1 for git diff)"
	efmsDoc      = `list of errorformat (https://github.com/haya14busa/errorformat)`
	fDoc         = `format name (run -list to see supported format name) for input. It's also used as tool name in review comment if -name is empty`
	listDoc      = `list supported pre-defined format names which can be used as -f arg`
	nameDoc      = `tool name in review comment. -f is used as tool name if -name is empty`
	ciDoc        = `CI service ('travis', 'circle-ci', 'droneio'(OSS 0.4) or 'common')

	GitHub/GitHub Enterprise:
		You need to set REVIEWDOG_GITHUB_API_TOKEN environment variable.
		Go to https://github.com/settings/tokens and create new Personal access token with repo scope.

		For GitHub Enterprise:
			export GITHUB_API="https://example.githubenterprise.com/api/v3"

		if you want to skip verifing SSL (please use this at your own risk)
			export REVIEWDOG_INSECURE_SKIP_VERIFY=true

	GitLab.com/self hosted Gitlab:
		You need to set REVIEWDOG_GITLAB_API_TOKEN environment variable.
		Go to https://gitlab.com/profile/personal_access_tokens 

		For self hosted GitLab:
			export GITLAB_API="https://example.gitlab.com/api/v4"

	"common" requires following environment variables
		CI_PULL_REQUEST	Pull Request number (e.g. 14)
		CI_COMMIT	SHA1 for the current build
		CI_REPO_OWNER	repository owner (e.g. "haya14busa" for https://github.com/haya14busa/reviewdog)
		CI_REPO_NAME	repository name (e.g. "reviewdog" for https://github.com/haya14busa/reviewdog)
`
	confDoc = `config file path`
)

var opt = &option{}

func init() {
	flag.BoolVar(&opt.version, "version", false, "print version")
	flag.StringVar(&opt.diffCmd, "diff", "", diffCmdDoc)
	flag.IntVar(&opt.diffStrip, "strip", 1, diffStripDoc)
	flag.Var(&opt.efms, "efm", efmsDoc)
	flag.StringVar(&opt.f, "f", "", fDoc)
	flag.BoolVar(&opt.list, "list", false, listDoc)
	flag.StringVar(&opt.name, "name", "", nameDoc)
	flag.StringVar(&opt.ci, "ci", "", ciDoc)
	flag.StringVar(&opt.conf, "conf", "", confDoc)
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
	if err := run(os.Stdin, os.Stdout, opt); err != nil {
		fmt.Fprintf(os.Stderr, "reviewdog: %v\n", err)
		os.Exit(1)
	}
}

func run(r io.Reader, w io.Writer, opt *option) error {
	ctx := context.Background()

	if opt.version {
		fmt.Fprintln(w, version)
		return nil
	}

	if opt.list {
		return runList(w)
	}

	// assume it's project based run when both -efm ane -f are not specified
	isProject := len(opt.efms) == 0 && opt.f == ""

	var cs reviewdog.CommentService
	var ds reviewdog.DiffService

	if isProject {
		cs = reviewdog.NewUnifiedCommentWriter(w)
	} else {
		cs = reviewdog.NewRawCommentWriter(w)
	}

	if opt.ci != "" {
		if os.Getenv("REVIEWDOG_GITHUB_API_TOKEN") != "" {
			gs, isPR, err := githubService(ctx, opt.ci)
			if err != nil {
				return err
			}
			if !isPR {
				fmt.Fprintf(os.Stderr, "this is not PullRequest build. CI: %v\n", opt.ci)
				return nil
			}
			cs = reviewdog.MultiCommentService(gs, cs)
			ds = gs
		} else if os.Getenv("REVIEWDOG_GITLAB_API_TOKEN") != "" {
			gs, isPR, err := gitlabService(ctx, opt.ci)
			if err != nil {
				return err
			}
			if !isPR {
				fmt.Fprintf(os.Stderr, "this is not MergeRequest build. CI: %v\n", opt.ci)
				return nil
			}
			cs = reviewdog.MultiCommentService(gs, cs)
			ds = gs
		} else {
			fmt.Fprintf(os.Stderr, "REVIEWDOG_GITHUB_API_TOKEN is not set\n")
			return nil
		}
	} else {
		// local
		d, err := diffService(opt.diffCmd, opt.diffStrip)
		if err != nil {
			return err
		}
		ds = d
	}

	if isProject {
		b, err := readConf(opt.conf)
		if err != nil {
			return fmt.Errorf("fail to open config: %v", err)
		}
		conf, err := project.Parse(b)
		if err != nil {
			return fmt.Errorf("config is invalid: %v", err)
		}
		return project.Run(ctx, conf, cs, ds)
	}

	p, err := reviewdog.NewParser(&reviewdog.ParserOpt{FormatName: opt.f, Errorformat: opt.efms})
	if err != nil {
		return fmt.Errorf("fail to create parser. use either -f or -efm: %v", err)
	}

	// tool name
	name := opt.name
	if name == "" && opt.f != "" {
		name = opt.f
	}

	app := reviewdog.NewReviewdog(name, p, cs, ds)
	return app.Run(ctx, r)
}

func runList(w io.Writer) error {
	tabw := tabwriter.NewWriter(w, 0, 8, 0, '\t', 0)
	for _, f := range sortedFmts(fmts.DefinedFmts()) {
		fmt.Fprintf(tabw, "%s\t%s\t- %s\n", f.Name, f.Description, f.URL)
	}
	fmt.Fprintf(tabw, "%s\t%s\t- %s\n", "checkstyle", "checkstyle XML format", "http://checkstyle.sourceforge.net/")
	return tabw.Flush()
}

type byFmtName []*fmts.Fmt

func (p byFmtName) Len() int           { return len(p) }
func (p byFmtName) Less(i, j int) bool { return p[i].Name < p[j].Name }
func (p byFmtName) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func sortedFmts(fs fmts.Fmts) []*fmts.Fmt {
	r := make([]*fmts.Fmt, 0, len(fs))
	for _, f := range fs {
		r = append(r, f)
	}
	sort.Sort(byFmtName(r))
	return r
}

func diffService(s string, strip int) (reviewdog.DiffService, error) {
	cmds, err := shellwords.Parse(s)
	if err != nil {
		return nil, err
	}
	if len(cmds) < 1 {
		return nil, errors.New("diff command is empty")
	}
	cmd := exec.Command(cmds[0], cmds[1:]...)
	d := reviewdog.NewDiffCmd(cmd, strip)
	return d, nil
}

func githubService(ctx context.Context, ci string) (githubservice *reviewdog.GitHubPullRequest, isPR bool, err error) {
	token, err := nonEmptyEnv("REVIEWDOG_GITHUB_API_TOKEN")
	if err != nil {
		return nil, isPR, err
	}
	var g *RequestInfo
	switch ci {
	case "travis":
		g, isPR, err = travis()
	case "circle-ci":
		g, isPR, err = circleci()
	case "droneio":
		g, isPR, err = droneio()
	case "common":
		g, isPR, err = commonci()
	default:
		return nil, isPR, fmt.Errorf("unsupported CI: %v", ci)
	}
	if err != nil {
		return nil, isPR, err
	}
	// TODO: support commit build
	if !isPR {
		return nil, isPR, nil
	}

	client, err := githubClient(ctx, token)
	if err != nil {
		return nil, isPR, err
	}

	githubservice, err = reviewdog.NewGitHubPullReqest(client, g.owner, g.repo, g.pr, g.sha)
	if err != nil {
		return nil, isPR, err
	}
	return githubservice, isPR, nil
}

func githubClient(ctx context.Context, token string) (*github.Client, error) {
	tr := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify()},
	}
	sslcli := &http.Client{Transport: tr}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, sslcli)
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	var err error
	client.BaseURL, err = githubBaseURL()
	return client, err
}

const defaultGitHubAPI = "https://api.github.com/"

func githubBaseURL() (*url.URL, error) {
	baseURL := os.Getenv("GITHUB_API")
	if baseURL == "" {
		baseURL = defaultGitHubAPI
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("GitHub base URL is invalid: %v, %v", baseURL, err)
	}
	return u, nil
}

func gitlabService(ctx context.Context, ci string) (gitlabservice *reviewdog.GitLabMergeRequest, isPR bool, err error) {
	token, err := nonEmptyEnv("REVIEWDOG_GITLAB_API_TOKEN")
	if err != nil {
		return nil, isPR, err
	}
	var g *RequestInfo
	switch ci {
	case "travis":
		g, isPR, err = travis()
	case "circle-ci":
		g, isPR, err = circleci()
	case "droneio":
		g, isPR, err = droneio()
	case "common":
		g, isPR, err = commonci()
	default:
		return nil, isPR, fmt.Errorf("unsupported CI: %v", ci)
	}
	if err != nil {
		return nil, isPR, err
	}
	// TODO: support commit build
	if !isPR {
		return nil, isPR, nil
	}

	client, err := gitlabClient(ctx, token)
	if err != nil {
		return nil, isPR, err
	}

	gitlabservice, err = reviewdog.NewGitLabMergeReqest(client, g.owner, g.repo, g.pr, g.sha)
	if err != nil {
		return nil, isPR, err
	}
	return gitlabservice, isPR, nil
}

func gitlabClient(_ context.Context, token string) (*gitlab.Client, error) {
	client := gitlab.NewClient(nil, token)
	var err error
	baseURL, err := gitlabBaseURL()
	client.SetBaseURL(baseURL.String())
	return client, err
}

const defaultGitLabAPI = "https://gitlab.com/api/v4"

func gitlabBaseURL() (*url.URL, error){
	baseURL := os.Getenv("GITLAB_API")
	if baseURL == "" {
		baseURL = defaultGitLabAPI
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("GitLab base URL is invalid: %v, %v", baseURL, err)
	}
	return u, nil

}

func insecureSkipVerify() bool {
	return os.Getenv("REVIEWDOG_INSECURE_SKIP_VERIFY") == "true"
}

func travis() (g *RequestInfo, isPR bool, err error) {
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

	g = &RequestInfo{
		owner: owner,
		repo:  repo,
		pr:    pr,
		sha:   sha,
	}
	return g, true, nil
}

// https://circleci.com/docs/environment-variables/
func circleci() (g *RequestInfo, isPR bool, err error) {
	var prs string // pull request number in string
	// For Pull Request from a same repository (CircleCI 2.0)
	// e.g. https: //github.com/haya14busa/reviewdog/pull/6
	// it might be better to support CI_PULL_REQUESTS instead.
	prs = os.Getenv("CIRCLE_PULL_REQUEST")
	if prs == "" {
		// For the backward compatibility with CircleCI 1.0.
		prs = os.Getenv("CI_PULL_REQUEST")
	}
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
	g = &RequestInfo{
		owner: owner,
		repo:  repo,
		pr:    pr,
		sha:   sha,
	}
	return g, true, nil
}

// http://readme.drone.io/usage/variables/
func droneio() (g *RequestInfo, isPR bool, err error) {
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

	owner, errOwner := nonEmptyEnv("DRONE_REPO_OWNER")
	repo, errRepo := nonEmptyEnv("DRONE_REPO_NAME")
	repoSlug, errSlug := nonEmptyEnv("DRONE_REPO")

	if (errOwner != nil || errRepo != nil) && errSlug != nil {
		return nil, true, fmt.Errorf("unable to detect repo and owner\n - %v\n - %v\n - %v", errOwner, errRepo, errSlug)
	}

	// Try to detect using env variable available in drone<=0.4
	if errSlug == nil {
		rss := strings.SplitN(repoSlug, "/", 2)
		if len(rss) < 2 {
			return nil, true, fmt.Errorf("unexpected env variable. DRONE_REPO=%v", repoSlug)
		}

		owner, repo = rss[0], rss[1]
	}

	sha, err := nonEmptyEnv("DRONE_COMMIT")
	if err != nil {
		return nil, true, err
	}
	g = &RequestInfo{
		owner: owner,
		repo:  repo,
		pr:    pr,
		sha:   sha,
	}
	return g, true, nil
}

func commonci() (g *RequestInfo, isPR bool, err error) {
	var prs string // pull request number in string
	prs = os.Getenv("CI_PULL_REQUEST")
	if prs == "" {
		// not a pull-request build
		return nil, false, nil
	}
	pr, err := strconv.Atoi(prs)
	if err != nil {
		return nil, true, fmt.Errorf("unexpected env variable (CI_PULL_REQUEST): %v", prs)
	}
	owner, err := nonEmptyEnv("CI_REPO_OWNER")
	if err != nil {
		return nil, true, err
	}
	repo, err := nonEmptyEnv("CI_REPO_NAME")
	if err != nil {
		return nil, true, err
	}
	sha, err := nonEmptyEnv("CI_COMMIT")
	if err != nil {
		return nil, true, err
	}
	g = &RequestInfo{
		owner: owner,
		repo:  repo,
		pr:    pr,
		sha:   sha,
	}
	return g, true, nil
}

// RequestInfo represents required information about GitHub PullRequest and Gitlab MergeRequest.
type RequestInfo struct {
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

func readConf(conf string) ([]byte, error) {
	var conffiles []string
	if conf != "" {
		conffiles = []string{conf}
	} else {
		conffiles = []string{
			".reviewdog.yaml",
			".reviewdog.yml",
			"reviewdog.yaml",
			"reviewdog.yml",
		}
	}
	for _, f := range conffiles {
		bytes, err := ioutil.ReadFile(f)
		if err == nil {
			return bytes, nil
		}
	}
	return nil, errors.New(".reviewdog.yml not found")
}
