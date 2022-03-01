package traceutils

import (
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

func CaptureException(c *gin.Context, err error) {
	if hub := sentrygin.GetHubFromContext(c); hub != nil {
		hub.CaptureException(err)
	}
}

func AddScopeTag(c *gin.Context, key, value string) {
	if hub := sentrygin.GetHubFromContext(c); hub != nil {
		hub.Scope().SetTag(key, value)
	}
}

func SetHandlerTag(c *gin.Context, handler string) {
	AddScopeTag(c, "handler", handler)
}
