package main

import (
	"net/http"

	"github.com/vvakame/sdlog/aelog"

	"github.com/reviewdog/reviewdog/doghouse/server/ciutil"
)

func warmupHandler(_ http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	aelog.Infof(ctx, "warming up server...")

	if err := ciutil.UpdateTravisCIIPAddrs(&http.Client{}); err != nil {
		aelog.Errorf(ctx, "failed to update travis CI IP addresses: %v", err)
	}

	aelog.Infof(ctx, "warmup done")
}
