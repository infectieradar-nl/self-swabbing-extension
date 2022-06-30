package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func HasValidAPIKey(validKeys []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := c.Request

		keysInHeader, ok := req.Header["Api-Key"]
		if !ok || len(keysInHeader) < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "A valid API key missing"})
			c.Abort()
			return
		}

		for _, k := range keysInHeader {
			for _, vk := range validKeys {
				if k == vk {
					c.Next()
					return
				}
			}
		}

		// If no keys matched:
		c.JSON(http.StatusBadRequest, gin.H{"error": "A valid API key missing"})
		c.Abort()
	}
}
