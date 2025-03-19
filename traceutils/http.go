package traceutils

import (
	"errors"
	"net/http"
	"net/http/httputil"

	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
)

// DumpRequest is a alias to dump http requests to string. It sends an error to
// sentry if a request is failed to parse
func DumpRequest(req *http.Request) string {
	dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		log.Error(errors.New("fail to dump request"), zap.Error(err))
	}

	return string(dump)
}

// DumpResponse is a alias to dump http responses to string. It sends an error to
// sentry if a response is failed to parse
func DumpResponse(resp *http.Response) string {
	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Error(errors.New("fail to dump response"), zap.Error(err))
	}

	return string(dump)
}
