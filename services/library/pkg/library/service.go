package library

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/tevino/abool"
	"go.uber.org/zap/zapcore"

	"github.com/MurmurationsNetwork/MurmurationsServices/pkg/core"
	"github.com/MurmurationsNetwork/MurmurationsServices/pkg/logger"
	"github.com/MurmurationsNetwork/MurmurationsServices/pkg/middleware/limiter"
	midlogger "github.com/MurmurationsNetwork/MurmurationsServices/pkg/middleware/logger"
	"github.com/MurmurationsNetwork/MurmurationsServices/pkg/mongo"
	"github.com/MurmurationsNetwork/MurmurationsServices/services/library/config"
	"github.com/MurmurationsNetwork/MurmurationsServices/services/library/global"
	"github.com/MurmurationsNetwork/MurmurationsServices/services/library/internal/controller/rest"
	"github.com/MurmurationsNetwork/MurmurationsServices/services/library/internal/repository/db"
	"github.com/MurmurationsNetwork/MurmurationsServices/services/library/internal/service"
)

func init() {
	global.Init()
}

// Service represents the library service.
type Service struct {
	// HTTP server
	server *http.Server
	// Atomic boolean to manage service state
	run *abool.AtomicBool
	// HTTP router for the library service
	router *gin.Engine
	// Ensures cleanup is only run once
	runCleanup sync.Once
	// Context for shutdown
	shutdownCtx context.Context
	// Cancel function for shutdown context
	shutdownCancelCtx context.CancelFunc
}

// NewService initializes a new library service.
func NewService() *Service {
	svc := &Service{
		run: abool.New(),
	}

	svc.setupServer()
	core.InstallShutdownHandler(svc.Shutdown)

	return svc
}

// setupServer configures and initializes the HTTP server.
func (s *Service) setupServer() {
	gin.SetMode(gin.ReleaseMode)
	s.router = gin.New()
	s.router.Use(s.middlewares()...)
	s.registerRoutes()

	if err := s.connectToMongoDB(); err != nil {
		s.panic("error when trying to connect to MongoDB", err)
	}

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%s", config.Conf.Server.Port),
		Handler:      s.router,
		ReadTimeout:  config.Conf.Server.TimeoutRead,
		WriteTimeout: config.Conf.Server.TimeoutWrite,
		IdleTimeout:  config.Conf.Server.TimeoutIdle,
	}

	s.shutdownCtx, s.shutdownCancelCtx = context.WithCancel(
		context.Background(),
	)
}

// connectToMongoDB establishes a connection to MongoDB.
func (s *Service) connectToMongoDB() error {
	uri := mongo.GetURI(
		config.Conf.Mongo.USERNAME,
		config.Conf.Mongo.PASSWORD,
		config.Conf.Mongo.HOST,
	)
	err := mongo.NewClient(uri, config.Conf.Mongo.DBName)
	if err != nil {
		return err
	}
	err = mongo.Client.Ping()
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) middlewares() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		gin.Recovery(),
		limiter.NewRateLimitWithOptions(limiter.RateLimitOptions{
			Period: config.Conf.Server.PostRateLimitPeriod,
			Method: "POST",
		}),
		limiter.NewRateLimitWithOptions(limiter.RateLimitOptions{
			Period: config.Conf.Server.GetRateLimitPeriod,
			Method: "GET",
		}),
		midlogger.NewLogger(),
		// CORS for all origins, allowing:
		// - GET and POST methods
		// - Origin, Authorization and Content-Type header
		// - Credentials share
		// - Preflight requests cached for 12 hours
		cors.New(cors.Config{
			AllowOrigins: []string{"*"},
			AllowHeaders: []string{
				"Origin",
				"Authorization",
				"Content-Type",
			},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}),
	}
}

// registerRoutes sets up the routes for the HTTP server.
func (s *Service) registerRoutes() {
	deprecationHandler := rest.NewDeprecationHandler()
	pingHandler := rest.NewPingHandler()
	schemaHandler := rest.NewSchemaHandler(
		service.NewSchemaService(db.NewSchemaRepo()),
	)
	countryHandler := rest.NewCountryHandler()

	v1 := s.router.Group("/v1")
	v1.Any("/*any", deprecationHandler.DeprecationV1)

	v2 := s.router.Group("/v2")
	v2.GET("/ping", pingHandler.Ping)
	v2.GET("/schemas", schemaHandler.Search)
	v2.GET("/schemas/:schemaName", schemaHandler.Get)
	v2.GET("/countries", countryHandler.GetMap)
}

// panic performs a cleanup and then emits the supplied message as the panic value.
func (s *Service) panic(msg string, err error, logFields ...zapcore.Field) {
	s.cleanup()
	logger.Panic(msg, err, logFields...)
}

// Run starts the library service and will block until the service is shutdown.
func (s *Service) Run() {
	s.run.Set()
	if err := s.server.ListenAndServe(); err != nil &&
		err != http.ErrServerClosed {
		s.panic("Error when trying to start the server", err)
	}
}

// WaitUntilUp returns a channel which blocks until the library service is up.
func (s *Service) WaitUntilUp() <-chan struct{} {
	initialized := make(chan struct{})
	go func() {
		for {
			resp, err := http.Get(
				fmt.Sprintf(
					"http://localhost:%s/v2/ping",
					config.Conf.Server.Port,
				),
			)
			if err == nil && resp.StatusCode == http.StatusOK {
				close(initialized)
				return
			}
			logger.Info(
				"Ping failed, waiting for service to finish starting...",
			)
			time.Sleep(time.Second)
		}
	}()
	return initialized
}

// Shutdown stops the library service.
func (s *Service) Shutdown() {
	if s.run.IsSet() {
		if err := s.server.Shutdown(s.shutdownCtx); err != nil {
			logger.Error("Library service shutdown failure", err)
		}
	}
	s.cleanup()
}

// cleanup will clean up the non-server resources associated with the service.
func (s *Service) cleanup() {
	s.runCleanup.Do(func() {
		s.shutdownCancelCtx()
		mongo.Client.Disconnect()
		logger.Info("Library service stopped gracefully")
	})
}
