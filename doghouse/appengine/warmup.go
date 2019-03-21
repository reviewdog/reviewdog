package main

import (
	"net/http"

	"github.com/haya14busa/reviewdog/doghouse/server/ciutil"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

func warmupHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	log.Infof(ctx, "warming up server...")

	if err := ciutil.UpdateTravisCIIPAddrs(urlfetch.Client(ctx)); err != nil {
		log.Errorf(ctx, "failed to update travis CI IP addresses: %v", err)
	}

	log.Infof(ctx, "warmup done")
}
