// +build go1.7

package nosurf

import (
	"context"
	"net/http"
)

func addNosurfContext(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), nosurfKey, &csrfContext{}))
}
