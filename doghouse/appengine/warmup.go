package main

import (
	"log"
	"net/http"

	"github.com/reviewdog/reviewdog/doghouse/server/ciutil"
)

func warmupHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] warming up server...\n")

	if err := ciutil.UpdateTravisCIIPAddrs(&http.Client{}); err != nil {
		log.Printf("[ERROR] failed to update travis CI IP addresses: %v\n", err)
	}

	log.Println("[INFO] warmup done")
}
