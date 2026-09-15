package main

import (
	"net/http"

	"github.com/vvakame/sdlog/aelog"
)

func warmupHandler(_ http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	aelog.Infof(ctx, "warming up server...")
	aelog.Infof(ctx, "warmup done")
}
