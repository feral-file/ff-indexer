package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TokenAuthenticate is the simplest authentication method based on a fixed key/value pair.
func TokenAuthenticate(tokenKey, tokenValue string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader(tokenKey)
		if token != tokenValue {
			abortWithError(c, http.StatusForbidden, "invalid api token", fmt.Errorf("invalid api token"))
			return
		}
		c.Next()
	}
}
