package middlewares

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/h-varmazyar/gatec/pkg/logger"
	jwtPkg "github.com/h-varmazyar/gopack/jwt"
)

type PublicAuthMiddleware struct {
}

func NewPublicAuthMiddleware() PublicAuthMiddleware {
	return PublicAuthMiddleware{}
}

func (p PublicAuthMiddleware) respondUnauthorized(c *gin.Context) {
	_ = c.Error(errors.New("unauthorized"))
	c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
}

func (p PublicAuthMiddleware) Handle(c *gin.Context) {
	authToken := c.GetHeader("Authorization")

	if authToken == "" {
		p.respondUnauthorized(c)
		return
	}
	authToken = strings.TrimSpace(strings.TrimPrefix(authToken, "Bearer"))
	claims, err := jwtPkg.ParseAuthClaims(authToken)
	if err != nil {
		p.respondUnauthorized(c)
		return
	}

	userClaim := claims.UserID
	if userClaim == "" {
		userClaim = claims.Subject
	}

	userID, err := strconv.Atoi(userClaim)
	if err != nil {
		p.respondUnauthorized(c)
		return
	}

	uID := uint(userID)
	logger.Logger.Infof("PublicAuthMiddleware: user_id=%d", uID)
	c.Set("user_id", uID)

	c.Next()
}
