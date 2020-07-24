package logger

import (
	"context"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/logging"
)

var lg *logging.Logger

// Init initizlize global logger for doghouse (reviewdog app server).
func Init(ctx context.Context) (close func() error) {
	client, err := logging.NewClient(ctx, os.Getenv("GOOGLE_CLOUD_PROJECT"))
	if err != nil {
		log.Fatal(err)
	}
	lg = client.Logger("reviewdog-global-logger")
	return client.Close
}

// LogWithReq logs message associated with given HTTP request.
func LogWithReq(r *http.Request, msg string) {
	lg.Log(logging.Entry{
		HTTPRequest: &logging.HTTPRequest{Request: r},
		Payload:     msg,
	})
}
