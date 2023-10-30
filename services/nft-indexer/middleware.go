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

// authenticate validates a JWT signed from account server. It uses the shared rsa public key
// to validate the token is signed from account server
// func (s *NFTIndexerServer) authenticate(c *gin.Context) {
// 	tokenStrings := strings.Split(c.GetHeader("Authorization"), " ")
// 	if len(tokenStrings) != 2 {
// 		abortWithError(c, http.StatusForbidden, "invalid authorization format", nil)
// 		return
// 	}

// 	authMethod := tokenStrings[0]
// 	if authMethod != "Bearer" {
// 		abortWithError(c, http.StatusForbidden, "invalid authorization format", nil)
// 		return
// 	}

// 	bearerToken := tokenStrings[1]
// 	var claims PlanJWTClaim
// 	token, err := jwt.ParseWithClaims(bearerToken, &claims, func(token *jwt.Token) (interface{}, error) {
// 		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
// 			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
// 		}

// 		return s.jwtPubkey, nil
// 	})

// 	if err != nil {
// 		abortWithError(c, http.StatusForbidden, "error invalid token", err)
// 		return
// 	}

// 	if !token.Valid {
// 		abortWithError(c, http.StatusForbidden, "error invalid token", nil)
// 		return
// 	}

// 	requester := claims.Subject
// 	if requester == "" {
// 		abortWithError(c, http.StatusForbidden, "error invalid requester", nil)
// 		return
// 	}
// 	c.Set("requester", requester)
// 	c.Set("plan", string(claims.Plan))
// 	c.Next()
// }
