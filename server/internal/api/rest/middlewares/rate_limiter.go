package middlewares

// import (
// 	"fmt"
// 	"sync"
// 	"time"

// 	"git.zarinline.ir/backend/core/internal/dexception"
// 	"github.com/gin-gonic/gin"
// )

// type RateLimiter struct {
// 	visitors       map[string]*visitor
// 	mu             sync.Mutex
// 	maxRequests    int
// 	windowDuration time.Duration
// }

// type visitor struct {
// 	requests []time.Time
// }

// func NewRateLimiter(maxRequests int, windowDuration time.Duration) *RateLimiter {
// 	return &RateLimiter{
// 		visitors:       make(map[string]*visitor),
// 		maxRequests:    maxRequests,
// 		windowDuration: windowDuration,
// 	}
// }

// func (rl *RateLimiter) getVisitor(ip string) *visitor {
// 	rl.mu.Lock()
// 	defer rl.mu.Unlock()

// 	if v, exists := rl.visitors[ip]; exists {

// 		now := time.Now()
// 		var validRequests []time.Time
// 		for _, requestTime := range v.requests {
// 			if now.Sub(requestTime) <= rl.windowDuration {
// 				validRequests = append(validRequests, requestTime)
// 			}
// 		}
// 		v.requests = validRequests
// 		return v
// 	}

// 	v := &visitor{}
// 	rl.visitors[ip] = v
// 	return v
// }

// func (rl *RateLimiter) Handle(c *gin.Context) {
// 	ip := c.ClientIP()
// 	v := rl.getVisitor(ip)

// 	now := time.Now()

// 	if len(v.requests) >= rl.maxRequests {

// 		_ = c.Error(dexception.NewRateLimiterException(fmt.Errorf("too many requests from IP: %s", ip)))
// 		c.Abort()
// 		return
// 	}

// 	v.requests = append(v.requests, now)

// 	c.Next()
// }
