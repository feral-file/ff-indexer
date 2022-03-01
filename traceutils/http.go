package traceutils

import (
	"net/http"
	"net/http/httputil"

	"github.com/getsentry/sentry-go"
)

// DumpRequest is a alias to dump http requests to string. It sends an error to
// sentry if a request is failed to parse
func DumpRequest(req *http.Request) string {
	dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		sentry.CaptureException(err)
	}

	return string(dump)
}

// DumpResponse is a alias to dump http responses to string. It sends an error to
// sentry if a response is failed to parse
func DumpResponse(resp *http.Response) string {
	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		sentry.CaptureException(err)
	}

	return string(dump)
}
