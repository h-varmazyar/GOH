package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"

	_ "github.com/h-varmazyar/gatec/docs" // swagger generated docs

	airdropHandlerV1 "github.com/h-varmazyar/gatec/internal/api/rest/handlers/v1/airdrop"
	applogHandlerV1 "github.com/h-varmazyar/gatec/internal/api/rest/handlers/v1/applog"
	authHandlerV1 "github.com/h-varmazyar/gatec/internal/api/rest/handlers/v1/auth"
	configurationHandlerV1 "github.com/h-varmazyar/gatec/internal/api/rest/handlers/v1/configuration"
	messagingHandlerV1 "github.com/h-varmazyar/gatec/internal/api/rest/handlers/v1/messaging"
	proxyHandlerV1 "github.com/h-varmazyar/gatec/internal/api/rest/handlers/v1/proxy"
	tabdealHandlerV1 "github.com/h-varmazyar/gatec/internal/api/rest/handlers/v1/tabdeal"
	tgAccountHandlerV1 "github.com/h-varmazyar/gatec/internal/api/rest/handlers/v1/tgaccount"
	tseMarketHandlerV1 "github.com/h-varmazyar/gatec/internal/api/rest/handlers/v1/tsemarket"
	userHandlerV1 "github.com/h-varmazyar/gatec/internal/api/rest/handlers/v1/user"
	"github.com/h-varmazyar/gatec/internal/api/rest/middlewares"
	"github.com/h-varmazyar/gatec/internal/consts"
	airdropService "github.com/h-varmazyar/gatec/internal/services/airdrop"
	applogService "github.com/h-varmazyar/gatec/internal/services/applog"
	authService "github.com/h-varmazyar/gatec/internal/services/auth"
	dynamicConfigService "github.com/h-varmazyar/gatec/internal/services/dynamicconfig"
	messagingService "github.com/h-varmazyar/gatec/internal/services/messaging"
	proxyService "github.com/h-varmazyar/gatec/internal/services/proxy"
	tgAccountService "github.com/h-varmazyar/gatec/internal/services/tgaccount"
	tseMarketService "github.com/h-varmazyar/gatec/internal/services/tsemarket"
	userService "github.com/h-varmazyar/gatec/internal/services/user"
	"github.com/h-varmazyar/gatec/pkg/logger"
)

type Server struct {
	engine *gin.Engine

	authHandler           authHandlerV1.Handler
	applicationLogHandler applogHandlerV1.Handler
	userHandler           userHandlerV1.Handler
	airdropHandler        airdropHandlerV1.Handler
	proxyHandler          proxyHandlerV1.Handler
	tabdealHandler        tabdealHandlerV1.Handler
	configurationHandler  configurationHandlerV1.Handler
	tgAccountHandler      tgAccountHandlerV1.Handler
	messagingHandler      messagingHandlerV1.Handler
	tseMarketHandler      tseMarketHandlerV1.Handler
}

func NewServer(
	appEnv consts.AppEnv,
	authService *authService.Service,
	applicationLogService *applogService.Service,
	userService *userService.Service,
	airdropService *airdropService.Service,
	proxyService *proxyService.Service,
	tgAccountService *tgAccountService.Service,
	messagingService *messagingService.Service,
	dynamicConfigService *dynamicConfigService.Service,
	tseMarketService *tseMarketService.Service,
) *Server {
	r := gin.New()
	r.ContextWithFallback = true

	r.RedirectTrailingSlash = false

	// if appEnv == consts.ProductionEnv {
	r.Use(middlewares.CORSMiddleware())
	// }
	// r.Use(middlewares.NewTrimmer().Trim(), middlewares.NewConvertToNull().ConvertToNull())
	r.Use(
		requestid.New(
			requestid.WithGenerator(func() string {
				return ""
			}),
			requestid.WithCustomHeaderStrKey("x-request-id"),
		),
	)

	server := &Server{
		authHandler:           authHandlerV1.New(authService),
		applicationLogHandler: applogHandlerV1.New(applicationLogService),
		userHandler:           userHandlerV1.New(userService),
		airdropHandler:        airdropHandlerV1.New(airdropService),
		proxyHandler:          proxyHandlerV1.New(proxyService),
		tabdealHandler:        tabdealHandlerV1.New(dynamicConfigService),
		configurationHandler:  configurationHandlerV1.New(dynamicConfigService),
		tgAccountHandler:      *tgAccountHandlerV1.New(tgAccountService),
		messagingHandler:      messagingHandlerV1.New(messagingService),
		tseMarketHandler:      tseMarketHandlerV1.New(tseMarketService),
		engine:                r,
	}
	return server
}

const headerTimeout = 10 * time.Second

func (s *Server) Serve(ctx context.Context, address string) error {
	s.setupAPIRoutes()
	srv := &http.Server{
		Addr:              address,
		Handler:           s.engine,
		ReadHeaderTimeout: headerTimeout,
	}

	logger.Logger.WithContext(ctx).Infof("rest server starting at: %s", address)
	if err := srv.ListenAndServe(); err != nil {
		return err
	}

	return nil
}
