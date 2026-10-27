package tabdeal

import (
	"fmt"
	"net/http"
	// "sync/atomic"
	"time"

	"github.com/h-varmazyar/gatec/configs"
)

type Adapter struct {
	httpClient *http.Client
	cfg        *configs.Tabdeal
	// authToken  atomic.Value
}

type AuthError struct {
	StatusCode int
	Body       string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("wallex auth error: status=%d body=%s", e.StatusCode, e.Body)
}

func NewAdapter(cfg *configs.Tabdeal) *Adapter {
	adapter := &Adapter{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
	// adapter.authToken.Store(cfg.AuthToken)
	return adapter
}

// func (a *Adapter) SetAuthToken(token string) {
// 	a.authToken.Store(token)
// }

// func (a *Adapter) getAuthToken() string {
// 	token, _ := a.authToken.Load().(string)
// 	return token
// }
