package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/haya14busa/reviewdog/doghouse"
	"github.com/haya14busa/reviewdog/doghouse/server"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

var githubAppsPrivateKey []byte

const (
	integrationID = 12131 // https://github.com/apps/reviewdog
)

func init() {
	// Private keys https://github.com/settings/apps/reviewdog
	const privateKeyFile = "./secret/github-apps.private-key.pem"
	var err error
	githubAppsPrivateKey, err = ioutil.ReadFile(privateKeyFile)
	if err != nil {
		log.Fatalf("could not read private key: %s", err)
	}
}

func main() {
	http.HandleFunc("/", handleTop)
	http.HandleFunc("/check", handleCheck)
	appengine.Main()
}

func handleTop(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "reviewdog")
}

func handleCheck(w http.ResponseWriter, r *http.Request) {
	var req doghouse.CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Fprintf(w, "failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	ctx := appengine.NewContext(r)
	dh, err := server.New(&req, githubAppsPrivateKey, integrationID, urlfetch.Client(ctx))
	if err != nil {
		fmt.Fprintln(w, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	res, err := dh.Check(ctx, &req)
	if err != nil {
		fmt.Fprintln(w, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		fmt.Fprintln(w, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}
