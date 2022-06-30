package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func HasValidInstanceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		instanceID := c.Param("instanceID")
		if instanceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "A valid InstanceID is missing"})
			c.Abort()
			return
		}

		c.Next()
	}
}
