package app

import (
	"context"
	"fmt"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/kholisrag/terraform-backend-gitops/pkg/config"
	"github.com/kholisrag/terraform-backend-gitops/pkg/logger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	tracer = otel.Tracer("terraform-backend-gitops")
)

func NewApp(config *config.Config) *gin.Engine {
	ctx := context.Background()

	switch config.Server.Mode {
	case "release", "prod", "production", "live":
		gin.SetMode(gin.ReleaseMode)
		logger.Info("Release mode enabled")
	case "local", "development", "dev", "test":
		gin.SetMode(gin.TestMode)
		logger.Info("Test mode enabled")
	default:
		gin.SetMode(gin.DebugMode)
		logger.Info("Debug mode enabled")
	}
	router := gin.New()

	if config.Tracing.Enabled {
		tp, err := initTracer(ctx, config)
		if err != nil {
			logger.Fatal("failed to initialize tracer", zap.Error(err))
		}
		defer func() {
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			if err := tp.Shutdown(ctx); err != nil {
				otel.Handle(err)
			}
		}()
	}

	// Integrate go-gin with opentelemetry
	router.Use(ginzap.Ginzap(logger.GetZapLogger(), time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(logger.GetZapLogger(), true))
	router.Use(otelgin.Middleware("terraform-backend-gitops"))

	router.GET("/healthz", func(c *gin.Context) {
		_, span := tracer.Start(c.Request.Context(), "healthz", oteltrace.WithAttributes(attribute.String("status", "ok")))
		defer span.End()
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	router.GET("/version", func(c *gin.Context) {
		_, span := tracer.Start(c.Request.Context(), "version", oteltrace.WithAttributes(attribute.String("version", config.Build.Version)))
		defer span.End()

		c.JSON(200, gin.H{
			"version": config.Build.Version,
			"commit":  config.Build.CommitHash,
			"build":   config.Build.BuildTime,
		})
	})

	routerGroupV1(config, &router.RouterGroup)

	return router
}

func initTracer(ctx context.Context, config *config.Config) (*sdktrace.TracerProvider, error) {
	// local exporter to stdout logs

	var exporter sdktrace.SpanExporter

	switch config.Tracing.Provider {
	case "stdout":
		exporter, err := stdout.New(stdout.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("failed to create %v exporter: %v", exporter, err)
		}
	case "otlptracehttp":
		// Create an OTLP exporter over gRPC
		exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithInsecure(), otlptracehttp.WithEndpoint(config.Tracing.OTLP.Endpoint))
		if err != nil {
			return nil, fmt.Errorf("failed to create %v exporter: %v, make sure to configure correct tracing.otlp.endpoint", exporter, err)
		} else {
			logger.Info("OTLP exporter created", zap.String("endpoint", config.Tracing.OTLP.Endpoint))
		}
	case "otlptracegrpc":
		// Create an OTLP exporter over gRPC
		exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure(), otlptracegrpc.WithEndpoint(config.Tracing.OTLP.Endpoint))
		if err != nil {
			return nil, fmt.Errorf("failed to create %v exporter: %v, make sure to configure correct tracing.otlp.endpoint", exporter, err)
		} else {
			logger.Info("otlp exporter created", zap.String("endpoint", config.Tracing.OTLP.Endpoint))
		}
	default:
		return nil, fmt.Errorf("unsupported tracing provider: %s", config.Tracing.Provider)
	}

	// Set sdktrace.Sampler dynamically based on config
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(config.Tracing.SampleRate))

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}
