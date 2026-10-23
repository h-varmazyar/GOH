package middlewares

// import (
// 	"encoding/base64"
// 	"errors"
// 	"strings"

// 	"github.com/gin-gonic/gin"
// 	"github.com/h-varmazyar/gatec/configs"
// )

// type BasicAuthMiddleware struct {
// 	cfg *configs.Config
// }

// func NewBasicAuthMiddleware(cfg *configs.Config) BasicAuthMiddleware {
// 	return BasicAuthMiddleware{cfg: cfg}
// }

// func (p BasicAuthMiddleware) respondUnauthorized(c *gin.Context) {
// 	_ = c.Error(dexception.NewUnauthorizedException(errors.New("unauthorized")))
// 	c.Abort()
// }

// func (p BasicAuthMiddleware) Handle(c *gin.Context) {
// 	authToken := c.GetHeader("Authorization")

// 	if authToken == "" {
// 		logger.Get().Debug(c.Request.Context(), "No Authorization header")
// 		p.respondUnauthorized(c)
// 		return
// 	}
// 	authType, credentials, ok := strings.Cut(authToken, " ")
// 	if !ok || authType != "Basic" {
// 		logger.Get().Debug(c.Request.Context(), "Invalid Authorization header")
// 		p.respondUnauthorized(c)
// 		return
// 	}

// 	decoded, err := base64.StdEncoding.DecodeString(credentials)
// 	if err != nil {
// 		logger.Get().Debug(c.Request.Context(), "Invalid encoded credentials")
// 		p.respondUnauthorized(c)
// 		return
// 	}

// 	userPass := strings.SplitN(string(decoded), ":", 2)
// 	if len(userPass) != 2 {
// 		logger.Get().Debug(c.Request.Context(), "Invalid auth len")
// 		p.respondUnauthorized(c)
// 		return
// 	}

// 	logger.Get().Debug(c.Request.Context(), "Authorization passed")

// 	username := userPass[0]
// 	password := userPass[1]

// 	if p.cfg.Auth.AdminUsername == username && p.cfg.Auth.AdminPassword == password {
// 		c.Next()
// 		return
// 	} else if p.cfg.VencyAdaptor.VencyThirdPartyAuthUsername == username && p.cfg.VencyAdaptor.VencyThirdPartyAuthPassword == password {
// 		c.Next()
// 		return
// 	}

// 	p.respondUnauthorized(c)

// }
