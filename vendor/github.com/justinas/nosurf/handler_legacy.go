// +build !go1.7

package nosurf

import "net/http"

func addNosurfContext(r *http.Request) *http.Request {
	return r
}
