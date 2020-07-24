package logger

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/logging"
	"go.opencensus.io/exporter/stackdriver/propagation"
)

var lg *logging.Logger

const loggerName = "reviewdog-global-logger"

var parent = "projects/" + os.Getenv("GOOGLE_CLOUD_PROJECT")

// Init initizlize global logger for doghouse (reviewdog app server).
func Init(ctx context.Context) (close func() error) {
	client, err := logging.NewClient(ctx, parent)
	if err != nil {
		log.Fatal(err)
	}
	lg = client.Logger(loggerName)
	return client.Close
}

// LogWithReq logs message associated with given HTTP request.
func LogWithReq(r *http.Request, msg string) {
	entry := logging.Entry{Payload: msg}
	sc, ok := (&propagation.HTTPFormat{}).SpanContextFromRequest(r)
	if ok {
		// Set SpanID and TraceID only instead of in http request.
		// Reference: https://github.com/googleapis/google-cloud-go/blob/cdaaf98f9226c39dc162b8e55083b2fbc67b4674/logging/logging.go#L878-L889
		entry.SpanID = sc.SpanID.String()
		if traceID := sc.TraceID.String(); traceID != "" {
			entry.Trace = fmt.Sprintf("%s/traces/%s", parent, traceID)
		}
	}
	lg.Log(entry)
}
