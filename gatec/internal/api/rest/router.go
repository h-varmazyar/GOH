package rest

import (
	"github.com/h-varmazyar/gatec/internal/api/rest/middlewares"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupAPIRoutes is a function that setup all routes for API
// @title           			gatec API collection
// @version         			1.0.0
// @description
// @Host 						localhost:8575
// @BasePath  					/
// @Schemes 					http
// @securityDefinitions.apikey	APIKeyAuth
// @in							header
// @name						X-API-Key
// @securityDefinitions.apikey	BearerTokenAuth
// @in							header
// @name						Authorization
func (s *Server) setupAPIRoutes() {
	r := s.engine

	// Swagger documentation route
	r.GET("/docs/api/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// r.GET("/health", healthCheckHandler.Check)

	v1 := r.Group("/v1")

	// auth handler
	{
		authHandlerV1 := v1.Group("/auth")
		authHandlerV1.POST("/register", s.authHandler.Register)
		authHandlerV1.POST("/login", s.authHandler.LoginByPassword)

		authHandlerV1.Use(middlewares.NewPublicAuthMiddleware().Handle)
		authHandlerV1.DELETE("/logout", s.authHandler.Logout)
	}

	{
		userHandlerV1 := v1.Group("/user")
		userHandlerV1.Use(middlewares.NewPublicAuthMiddleware().Handle)
		userHandlerV1.GET("/info", s.userHandler.UserInfo)
	}
	if s.airdropHandler.HasService() {
		airdropHandlerV1 := v1.Group("/airdrops")
		airdropHandlerV1.Use(middlewares.NewPublicAuthMiddleware().Handle)
		airdropHandlerV1.GET("/gifts", s.airdropHandler.ListGifts)
		airdropHandlerV1.GET("/gifts/:id", s.airdropHandler.GetGift)
		airdropHandlerV1.POST("/gifts/:id/retry", s.airdropHandler.RetryCollectGift)
		airdropHandlerV1.GET("/routes", s.airdropHandler.ListRoutes)
		airdropHandlerV1.POST("/routes", s.airdropHandler.CreateRoute)
	}

	if s.applicationLogHandler.HasService() {
		logGroup := v1.Group("/logs")
		logGroup.Use(middlewares.NewPublicAuthMiddleware().Handle)
		logGroup.GET("/:component", s.applicationLogHandler.List)
	}

	if s.configurationHandler.HasService() {
		configGroup := v1.Group("/config")
		configGroup.Use(middlewares.NewPublicAuthMiddleware().Handle)
		configGroup.GET("/sections", s.configurationHandler.ListSections)
		// configGroup.PATCH("/sections/:section", s.configurationHandler.UpdateSection)

		// workerGroup := v1.Group("/workers")
		// workerGroup.Use(middlewares.NewPublicAuthMiddleware().Handle)
		// workerGroup.GET("", s.configurationHandler.ListWorkers)
		// workerGroup.PUT("/:name/enabled", s.configurationHandler.SetWorkerEnabled)
	}

	if s.tabdealHandler.HasService() {
		// tabdealHandlerV1 := v1.Group("/tabdeal")
		// tabdealHandlerV1.Use(middlewares.NewPublicAuthMiddleware().Handle)
		// tabdealHandlerV1.PATCH("/auth-token", s.tabdealHandler.UpdateAuthToken)
	}

	if s.tgAccountHandler.HasService() {
		tgAccountHandler := v1.Group("telegram")
		tgAccountHandler.POST("/sessions", s.tgAccountHandler.CreateAccount)
		tgAccountHandler.POST("/sessions/phone", s.tgAccountHandler.SubmitPhone)
		tgAccountHandler.POST("/sessions/code", s.tgAccountHandler.SubmitCode)
		tgAccountHandler.POST("/sessions/password", s.tgAccountHandler.SubmitPassword)
		tgAccountHandler.POST("/sessions/re-auth", s.tgAccountHandler.ReAuthenticate)
		tgAccountHandler.GET("/sessions/:id/state", s.tgAccountHandler.GetState)
		tgAccountHandler.GET("/accounts", s.tgAccountHandler.ListAccounts)
		tgAccountHandler.POST("/subscription-filters", s.tgAccountHandler.CreateSubscriptionFilter)
		tgAccountHandler.GET("/subscription-filters", s.tgAccountHandler.ListSubscriptionFilters)
		tgAccountHandler.DELETE("/subscription-filters/:id", s.tgAccountHandler.DeleteSubscriptionFilter)

		// Proxy management routes
		if s.proxyHandler.HasService() {
			proxyHandler := tgAccountHandler.Group("/proxies")
			proxyHandler.Use(middlewares.NewPublicAuthMiddleware().Handle)
			proxyHandler.GET("", s.proxyHandler.ListProxies)
			proxyHandler.GET("/active", s.proxyHandler.ListActiveProxies)
			proxyHandler.GET("/enabled", s.proxyHandler.ListEnabledProxies)
			proxyHandler.POST("", s.proxyHandler.CreateProxy)
			proxyHandler.GET("/stats", s.proxyHandler.GetStats)

			proxyHandler.GET("/:id", s.proxyHandler.GetProxy)
			proxyHandler.PUT("/:id", s.proxyHandler.UpdateProxy)
			proxyHandler.PATCH("/:id/toggle", s.proxyHandler.ToggleProxy)
			proxyHandler.DELETE("/:id", s.proxyHandler.DeleteProxy)
			proxyHandler.POST("/:id/force-check", s.proxyHandler.ForceCheck)
		}

		// Messaging routes (bidirectional messaging)
		if s.messagingHandler.HasService() {
			msgGroup := tgAccountHandler.Group("/messaging")
			msgGroup.Use(middlewares.NewPublicAuthMiddleware().Handle)

			// Message sending
			msgGroup.POST("/send", s.messagingHandler.SendMessage)
			msgGroup.POST("/schedule", s.messagingHandler.ScheduleMessage)
			msgGroup.DELETE("/schedule/:id", s.messagingHandler.CancelScheduledMessage)

			// Access control
			msgGroup.GET("/access/rules", s.messagingHandler.ListAccessRules)
			msgGroup.POST("/access/rules", s.messagingHandler.GrantAccess)
			msgGroup.DELETE("/access/rules/:id", s.messagingHandler.RevokeAccess)
			msgGroup.GET("/access/check", s.messagingHandler.CheckAccess)

			// Senders
			msgGroup.GET("/senders", s.messagingHandler.ListSenders)
		}
	}
}
