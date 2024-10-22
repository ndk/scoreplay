package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/alexliesenfeld/health"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/redis/rueidis"
	"golang.org/x/sync/errgroup"

	"scoreplay/internal/handlers"
	"scoreplay/internal/logger"
	"scoreplay/internal/middleware"
	"scoreplay/internal/service"
	"scoreplay/internal/signal"
	"scoreplay/pkg/api"
)

type Config struct {
	Logger logger.Config `env:"LOGGER"`
	AWS    struct {
		EndpointURL string `env:"ENDPOINT_URL" default:""`
		S3          struct {
			UsePathStyle bool `env:"USE_PATH_STYLE"`
		} `env:"S3"`
	} `env:"AWS"`
	Redis struct {
		InitAddress  []string `env:"INIT_ADDRESS,required"`
		Username     string   `env:"USERNAME,required"`
		Password     string   `env:"PASSWORD,required"`
		SelectDB     int      `env:"SELECT_DB,required"`
		DisableCache bool     `env:"DISABLE_CACHE" default:"false"`
	} `env:"REDIS"`
	Storage struct {
		Bucket string `env:"BUCKET,required"`
	} `env:"STORAGE"`
	Server struct {
		Address           string        `env:"ADDRESS" default:":8080"`
		ShutdownTimeout   time.Duration `env:"SHUTDOWN_TIMEOUT" default:"30s"`
		ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT" default:"5s"`
	} `env:"SERVER"`
	Healthcheck struct {
		CacheDuration time.Duration `env:"CACHE_DURATION" default:"1s"`
		Timeout       time.Duration `env:"TIMEOUT" default:"10s"`
	} `env:"HEALTHCHECK"`
}

func Run(ctx context.Context, cfg Config) error {
	client, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  cfg.Redis.InitAddress,
		Username:     cfg.Redis.Username,
		Password:     cfg.Redis.Password,
		SelectDB:     cfg.Redis.SelectDB,
		DisableCache: cfg.Redis.DisableCache,
	})
	if err != nil {
		return fmt.Errorf("creating redis client: %w", err)
	}
	defer client.Close()

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading aws config: %w", err)
	}
	presignClient := s3.NewPresignClient(s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		// https://github.com/aws/aws-sdk-go-v2/discussions/2578
		if cfg.AWS.S3.UsePathStyle {
			o.UsePathStyle = true
		}
	}))

	endpointURL, err := url.Parse(cfg.AWS.EndpointURL)
	if err != nil {
		return fmt.Errorf("parsing endpoint url: %w", err)
	}
	qs := service.NewMediaService(client, presignClient, *endpointURL, cfg.Storage.Bucket)

	swagger, err := api.GetSwagger()
	if err != nil {
		return fmt.Errorf("getting swagger: %w", err)
	}
	swagger.Servers = nil

	r := http.NewServeMux()
	r.Handle("/health", health.NewHandler(health.NewChecker(
		health.WithCacheDuration(cfg.Healthcheck.CacheDuration),
		health.WithTimeout(cfg.Healthcheck.Timeout),
	)))
	api.HandlerWithOptions(api.NewStrictHandler(handlers.NewMediaAPI(qs), nil), api.StdHTTPServerOptions{
		BaseRouter: r,
		Middlewares: []api.MiddlewareFunc{
			middleware.RecoveryMiddleware,
			nethttpmiddleware.OapiRequestValidator(swagger),
		},
	})

	g, ctx := errgroup.WithContext(ctx)

	srv := http.Server{
		Handler:           r,
		Addr:              cfg.Server.Address,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
	}

	g.Go(func() error { return signal.WaitForSignal(ctx) })
	g.Go(func() error {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	g.Go(func() error {
		<-ctx.Done()
		srv.SetKeepAlivesEnabled(false)
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		return srv.Shutdown(ctx) //nolint: contextcheck
	})

	return g.Wait()
}
