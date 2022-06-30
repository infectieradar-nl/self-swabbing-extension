package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequirePayload blocks post requests that have no payload attached
func RequirePayload() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "payload missing"})
			return
		}
		c.Next()
	}
}
