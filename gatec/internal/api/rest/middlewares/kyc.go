package middlewares

// import (
// 	"errors"

// 	"git.zarinline.ir/backend/core/internal/dexception"
// 	"git.zarinline.ir/backend/core/internal/services/user"
// 	"github.com/gin-gonic/gin"
// )

// type KycMiddleware struct {
// 	userService user.Service
// }

// func NewKycMiddleware(userService user.Service) KycMiddleware {
// 	return KycMiddleware{
// 		userService: userService,
// 	}
// }

// func (k KycMiddleware) respondInvalidKyc(c *gin.Context) {
// 	_ = c.Error(dexception.NewInvalidKycException(errors.New("forbidden")))
// 	c.Abort()
// }

// func (k KycMiddleware) Handle(c *gin.Context) {
// 	userID := c.GetUint("user_id")
// 	if userID == 0 {
// 		k.respondInvalidKyc(c)
// 		return
// 	}

// 	userInfo, err := k.userService.Info(c, userID)
// 	if err != nil {
// 		k.respondInvalidKyc(c)
// 		return
// 	}

// 	if !userInfo.Kyc {
// 		k.respondInvalidKyc(c)
// 		return
// 	}

// 	c.Next()
// }
