package main

import (
	"net/http"

	"github.com/reviewdog/reviewdog/doghouse/server/ciutil"
	"github.com/vvakame/sdlog/aelog"
)

func warmupHandler(_ http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	aelog.Infof(ctx, "warming up server...\n")

	if err := ciutil.UpdateTravisCIIPAddrs(&http.Client{}); err != nil {
		aelog.Errorf(ctx, "failed to update travis CI IP addresses: %v\n", err)
	}

	aelog.Infof(ctx, "warmup done")
}
