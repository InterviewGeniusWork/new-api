package middleware

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

// InternalAuth ensures only trusted backends can access internal routes.
func InternalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.IGInternalSecret == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "internal api disabled",
			})
			return
		}
		if c.GetHeader("X-IG-Internal-Key") != common.IGInternalSecret {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "unauthorized",
			})
			return
		}
		c.Next()
	}
}
