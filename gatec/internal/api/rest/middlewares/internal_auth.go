package middlewares

// import (
// 	"net/http"

// 	"git.zarinline.ir/backend/core/configs"
// 	"git.zarinline.ir/backend/pkg/zar-logger/logger"

// 	"github.com/gin-gonic/gin"
// )

// type InternalAuthMiddleware struct {
// 	cfg *configs.Config
// }

// func NewInternalAuthMiddleware(cfg *configs.Config) InternalAuthMiddleware {
// 	return InternalAuthMiddleware{
// 		cfg: cfg,
// 	}
// }

// func (i InternalAuthMiddleware) respondUnauthorized(c *gin.Context) {
// 	logger.Get().Error(c.Request.Context(), "failed to retrieve X-Internal-Auth information", nil)
// 	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
// }

// func (i InternalAuthMiddleware) Handle(c *gin.Context) {
// 	auth := c.GetHeader("X-Internal-Auth")

// 	if auth == "" {
// 		i.respondUnauthorized(c)
// 		return
// 	}

// 	if auth != i.cfg.InternalAuthToken {
// 		i.respondUnauthorized(c)
// 		return
// 	}

// 	c.Next()
// }
