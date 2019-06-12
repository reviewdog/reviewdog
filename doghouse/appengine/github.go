package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-github/v26/github"
	"github.com/justinas/nosurf"
	"github.com/reviewdog/reviewdog/doghouse/server"
	"github.com/reviewdog/reviewdog/doghouse/server/cookieman"
	"github.com/reviewdog/reviewdog/doghouse/server/storage"
	"golang.org/x/oauth2"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

type GitHubHandler struct {
	clientID     string
	clientSecret string

	tokenStore     *cookieman.CookieStore
	redirURLStore  *cookieman.CookieStore // Redirect URL after login.
	authStateStore *cookieman.CookieStore

	repoTokenStore storage.GitHubRepositoryTokenStore

	privateKey    []byte
	integrationID int
}

func NewGitHubHandler(clientID, clientSecret string, c *cookieman.CookieMan, privateKey []byte, integrationID int) *GitHubHandler {
	return &GitHubHandler{
		clientID:       clientID,
		clientSecret:   clientSecret,
		tokenStore:     c.NewCookieStore("github-token", nil),
		redirURLStore:  c.NewCookieStore("github-redirect-url", nil),
		authStateStore: c.NewCookieStore("github-auth-state", nil),
		repoTokenStore: &storage.GitHubRepoTokenDatastore{},
		integrationID:  integrationID,
		privateKey:     privateKey,
	}
}

type ghTopTmplData struct {
	Title string
	User  tmplUser

	App struct {
		Name    string
		HTMLURL string
	}

	Installations []tmplInstallation
}

type tmplInstallation struct {
	Account        string
	AccountHTMLURL string
	AccountIconURL string
	HTMLURL        string
}

type ghRepoTmplData struct {
	Title     string
	Token     string
	User      tmplUser
	Repo      tmplRepo
	CSRFToken string
}

type tmplUser struct {
	Name      string
	IconURL   string
	GitHubURL string
}

type tmplRepo struct {
	Owner     string
	Name      string
	GitHubURL string
}

func (g *GitHubHandler) buildGithubAuthURL(r *http.Request, state string) string {
	redirURL := *r.URL
	redirURL.Path = "/gh/_auth/callback"
	redirURL.RawQuery = ""
	redirURL.Fragment = ""
	const baseURL = "https://github.com/login/oauth/authorize"
	authURL := fmt.Sprintf("%s?client_id=%s&redirect_url=%s&state=%s",
		baseURL, g.clientID, redirURL.RequestURI(), state)
	return authURL
}

func (g *GitHubHandler) HandleAuthCallback(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	code, state := r.FormValue("code"), r.FormValue("state")
	if code == "" || state == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "code and state param is empty")
		return
	}

	// Verify state.
	cookieState, err := g.authStateStore.Get(r)
	if err != nil || state != string(cookieState) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "state is invalid")
		return
	}
	g.authStateStore.Clear(w)

	// Request and save access token.
	token, err := g.requestAccessToken(ctx, code, state)
	if err != nil {
		log.Errorf(ctx, "failed to get access token: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "failed to get GitHub access token")
		return
	}
	g.tokenStore.Set(w, []byte(token))

	// Redirect.
	redirURL := "/gh/"
	if r, _ := g.redirURLStore.Get(r); err == nil {
		redirURL = string(r)
		g.redirURLStore.Clear(w)
	}
	http.Redirect(w, r, redirURL, http.StatusFound)
}

func (g *GitHubHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	g.tokenStore.Clear(w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (g *GitHubHandler) LogInHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)
		if g.isLoggedIn(r) {
			h.ServeHTTP(w, r)
			return
		}
		// Not logged in yet.
		log.Debugf(ctx, "Not logged in yet.")
		state := securerandom(16)
		g.redirURLStore.Set(w, []byte(r.URL.RequestURI()))
		g.authStateStore.Set(w, []byte(state))
		http.Redirect(w, r, g.buildGithubAuthURL(r, state), http.StatusFound)
	})
}

func (g *GitHubHandler) isLoggedIn(r *http.Request) bool {
	ok, _ := g.token(r)
	return ok
}

func securerandom(n int) string {
	b := make([]byte, n)
	io.ReadFull(rand.Reader, b[:])
	return fmt.Sprintf("%x", b)
}

// https://developer.github.com/apps/building-github-apps/identifying-and-authorizing-users-for-github-apps/#2-users-are-redirected-back-to-your-site-by-github
// POST https://github.com/login/oauth/access_token
func (g *GitHubHandler) requestAccessToken(ctx context.Context, code, state string) (string, error) {
	const u = "https://github.com/login/oauth/access_token"
	cli := urlfetch.Client(ctx)
	data := url.Values{}
	data.Set("client_id", g.clientID)
	data.Set("client_secret", g.clientSecret)
	data.Set("code", code)
	data.Set("state", state)

	req, err := http.NewRequest(http.MethodPost, u, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req = req.WithContext(ctx)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Accept", "application/vnd.github.machine-man-preview+json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := cli.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request access token: %v", err)
	}
	defer res.Body.Close()

	b, _ := ioutil.ReadAll(res.Body)

	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&token); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	if token.AccessToken == "" {
		log.Errorf(ctx, "response doesn't contain token (resopnse: %s)", b)
		return "", errors.New("response doesn't contain GitHub access token")
	}

	return token.AccessToken, nil
}

func (g *GitHubHandler) token(r *http.Request) (bool, string) {
	b, err := g.tokenStore.Get(r)
	if err != nil {
		return false, ""
	}
	return true, string(b)
}

func (g *GitHubHandler) HandleGitHubTop(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	ok, token := g.token(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	ghcli := github.NewClient(NewAuthClient(ctx, urlfetch.Client(ctx).Transport, ts))

	// /gh/{owner}/{repo}
	paths := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	switch len(paths) {
	case 1:
		g.handleTop(ctx, ghcli, w, r)
	case 3:
		g.handleRepo(ctx, ghcli, w, r, paths[1], paths[2])
	default:
		notfound(w)
	}
}

func notfound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintln(w, "404 Not Found")
}

func (g *GitHubHandler) getUserOrBadRequest(ctx context.Context, ghcli *github.Client, w http.ResponseWriter) (bool, *github.User) {
	u, _, err := ghcli.Users.Get(ctx, "")
	if err != nil {
		// Token seeims invalid. Clear it before returning BadRequest status.
		g.tokenStore.Clear(w)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Cannot get GitHub authenticated user. Please reload the page again.")
		return false, nil
	}
	return true, u
}

func (g *GitHubHandler) handleTop(ctx context.Context, ghcli *github.Client, w http.ResponseWriter, r *http.Request) {
	ok, u := g.getUserOrBadRequest(ctx, ghcli, w)
	if !ok {
		return
	}

	data := &ghTopTmplData{
		Title: "GitHub - reviewdog",
		User: tmplUser{
			Name:      u.GetName(),
			IconURL:   u.GetAvatarURL(),
			GitHubURL: u.GetHTMLURL(),
		},
	}

	ghAppCli, err := server.NewGitHubClient(ctx, &server.NewGitHubClientOption{
		Client:        urlfetch.Client(ctx),
		IntegrationID: g.integrationID,
		PrivateKey:    g.privateKey,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err)
		return
	}
	app, _, err := ghAppCli.Apps.Get(ctx, "")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err)
		return
	}
	data.App.Name = app.GetName()
	data.App.HTMLURL = app.GetHTMLURL()

	installations, _, err := ghcli.Apps.ListUserInstallations(ctx, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err)
		return
	}
	for _, inst := range installations {
		data.Installations = append(data.Installations, tmplInstallation{
			Account:        inst.GetAccount().GetLogin(),
			AccountHTMLURL: inst.GetAccount().GetHTMLURL(),
			AccountIconURL: inst.GetAccount().GetAvatarURL(),
			HTMLURL:        inst.GetHTMLURL(),
		})
	}

	ghTopTmpl.ExecuteTemplate(w, "base", data)
}

func (g *GitHubHandler) handleRepo(ctx context.Context, ghcli *github.Client, w http.ResponseWriter, r *http.Request, owner, repoName string) {
	repo, _, err := ghcli.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		if err, ok := err.(*github.ErrorResponse); ok {
			if err.Response.StatusCode == http.StatusNotFound {
				notfound(w)
				return
			}
		}
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "failed to get repo: %#v", err)
		return
	}

	if !repo.GetPermissions()["push"] {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "You don't have write permission for %s.", repo.GetHTMLURL())
		return
	}

	ok, u := g.getUserOrBadRequest(ctx, ghcli, w)
	if !ok {
		return
	}

	// Regenerate Token.
	if r.Method == http.MethodPost {
		if _, err := server.RegenerateRepoToken(ctx, g.repoTokenStore, repo.Owner.GetLogin(), repo.GetName(), repo.GetID()); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "failed to update repository token: %v", err)
			return
		}
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
	}

	repoToken, err := server.GetOrGenerateRepoToken(ctx, g.repoTokenStore, repo.Owner.GetLogin(), repo.GetName(), repo.GetID())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "failed to get repository token for %s.", repo.GetHTMLURL())
		return
	}

	ghRepoTmpl.ExecuteTemplate(w, "base", &ghRepoTmplData{
		Title: fmt.Sprintf("%s/%s - reviewdog", repo.Owner.GetLogin(), repo.GetName()),
		Token: repoToken,
		User: tmplUser{
			Name:      u.GetName(),
			IconURL:   u.GetAvatarURL(),
			GitHubURL: u.GetHTMLURL(),
		},
		Repo: tmplRepo{
			Owner:     repo.Owner.GetLogin(),
			Name:      repo.GetName(),
			GitHubURL: repo.GetHTMLURL(),
		},
		CSRFToken: nosurf.Token(r),
	})
}

func NewAuthClient(ctx context.Context, base http.RoundTripper, token oauth2.TokenSource) *http.Client {
	tc := oauth2.NewClient(ctx, token)
	tr := tc.Transport.(*oauth2.Transport)
	tr.Base = base
	return tc
}
