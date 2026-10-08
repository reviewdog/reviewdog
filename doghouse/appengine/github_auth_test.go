package main

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestBuildGithubAuthURLCallbackPath(t *testing.T) {
	h := &GitHubHandler{clientID: "client+id"}
	r := httptest.NewRequest("GET", "https://reviewdog.example/gh/owner/repo?foo=bar", nil)
	u, err := url.Parse(h.buildGithubAuthURL(r, "state&value"))
	if err != nil {
		t.Fatal(err)
	}
	if got := u.Query().Get("redirect_uri"); got != "https://reviewdog.example/gh_/auth/callback" {
		t.Fatalf("callback URL = %q, want registered /gh_/auth/callback", got)
	}
	if got := u.Query().Get("state"); got != "state&value" {
		t.Fatalf("state = %q, want state&value", got)
	}
	if got := u.Query().Get("client_id"); got != "client+id" {
		t.Fatalf("client_id = %q, want client+id", got)
	}
}
