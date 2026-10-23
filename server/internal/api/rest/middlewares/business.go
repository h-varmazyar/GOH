package middlewares

// import (
// 	"errors"

// 	"git.zarinline.ir/backend/core/internal/services/user"
// 	"git.zarinline.ir/backend/pkg/essential/exception"
// 	"github.com/gin-gonic/gin"
// )

// type BusinessMiddleware struct {
// 	userService user.Service
// }

// func NewBusinessMiddleware(userService user.Service) BusinessMiddleware {
// 	return BusinessMiddleware{
// 		userService: userService,
// 	}
// }

// func (k BusinessMiddleware) respondInvalidBusiness(c *gin.Context) {
// 	_ = c.Error(exception.NewForbiddenError(errors.New("forbidden"), nil, nil))
// 	c.Abort()
// }

// func (k BusinessMiddleware) Handle(c *gin.Context) {
// 	userID := c.GetUint("user_id")
// 	if userID == 0 {
// 		k.respondInvalidBusiness(c)
// 		return
// 	}

// 	_, err := k.userService.GetBusinessByUserID(c, userID)
// 	if err != nil {
// 		k.respondInvalidBusiness(c)
// 		return
// 	}

// 	c.Next()
// }
