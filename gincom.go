package gincom

import (
	"fmt"
	"github.com/bugsnag/bugsnag-go-gin"
	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"net/http"
	"os"
	"time"
)

func BootstrapGin() {
	// load .env file values
	err := godotenv.Load()
	if err != nil {
		log.Info().Msg("couldn't find .env file, using os environment variables")
	}

	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

type BugsnagOptions struct {
	APIKey          string
	AppVersion      string
	ProjectPackages []string
}

type MiddlewareOptions struct {
	Coors    bool
	Recovery bool
	Zerolog  bool
}

type EngineOptions struct {
	BugsnagOptions    bugsnag.Configuration
	MiddlewareOptions MiddlewareOptions
	ReleaseMode       string
	EnableHealthCheck bool
}

func DefaultEngineOptions() *EngineOptions {
	return &EngineOptions{
		MiddlewareOptions: MiddlewareOptions{
			Zerolog:  true,
			Coors:    true,
			Recovery: true,
		},
		BugsnagOptions: bugsnag.Configuration{
			APIKey:     os.Getenv("BUGSNAG_API_KEY"),
			AppVersion: os.Getenv("API_VERSION"),
		},
		ReleaseMode:       gin.ReleaseMode,
		EnableHealthCheck: true,
	}
}

func GinEngine(o *EngineOptions) *gin.Engine {
	gin.SetMode(o.ReleaseMode)

	// create router
	r := gin.New()

	// setup logging
	if o.MiddlewareOptions.Zerolog {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		r.Use(logger.SetLogger(logger.WithSkipPath([]string{"/health"})))
	}

	// setup recovery
	if o.MiddlewareOptions.Recovery {
		r.Use(gin.Recovery())
	}

	// setup bugsnag
	if o.BugsnagOptions.APIKey != "" {
		log.Info().Msg("bugsnag enabled")
		r.Use(bugsnaggin.AutoNotify(o.BugsnagOptions))
	}

	// setup coors
	if o.MiddlewareOptions.Coors {
		r.Use(cors.Default())
	}

	// setup health-check
	if o.EnableHealthCheck {
		r.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "w00t"})
		})
	}

	return r
}

func DefaultGinEngine() *gin.Engine {
	return GinEngine(DefaultEngineOptions())
}

type HttpServerOptions struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func HttpServer(r *gin.Engine, o HttpServerOptions) *http.Server {
	log.Info().Msg(fmt.Sprintf("starting server on port %d", o.Port))
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", o.Port),
		ReadTimeout:  o.ReadTimeout,
		WriteTimeout: o.WriteTimeout,
		Handler:      r,
	}
}

func NewHttpServer(r *gin.Engine) *http.Server {
	opts := HttpServerOptions{
		Port:         3141,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return HttpServer(r, opts)
}

func HttpListen(f func(*gin.Engine)) {
	e := DefaultGinEngine()

	f(e)

	s := NewHttpServer(e)

	if err := s.ListenAndServe(); err != nil {
		log.Error().Err(err).Msg("failed to start gin")
	}
}
