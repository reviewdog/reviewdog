// +build go1.7

package nosurf

import "net/http"

type ctxKey int

const (
	nosurfKey ctxKey = iota
)

type csrfContext struct {
	// The masked, base64 encoded token
	// That's suitable for use in form fields, etc.
	token string
	// reason for the failure of CSRF check
	reason error
}

// Token takes an HTTP request and returns
// the CSRF token for that request
// or an empty string if the token does not exist.
//
// Note that the token won't be available after
// CSRFHandler finishes
// (that is, in another handler that wraps it,
// or after the request has been served)
func Token(req *http.Request) string {
	ctx, ok := req.Context().Value(nosurfKey).(*csrfContext)
	if !ok {
		return ""
	}

	return ctx.token
}

// Reason takes an HTTP request and returns
// the reason of failure of the CSRF check for that request
//
// Note that the same availability restrictions apply for Reason() as for Token().
func Reason(req *http.Request) error {
	ctx := req.Context().Value(nosurfKey).(*csrfContext)

	return ctx.reason
}

func ctxClear(_ *http.Request) {
}

func ctxSetToken(req *http.Request, token []byte) {
	ctx := req.Context().Value(nosurfKey).(*csrfContext)
	ctx.token = b64encode(maskToken(token))
}

func ctxSetReason(req *http.Request, reason error) {
	ctx := req.Context().Value(nosurfKey).(*csrfContext)
	if ctx.token == "" {
		panic("Reason should never be set when there's no token in the context yet.")
	}

	ctx.reason = reason
}
