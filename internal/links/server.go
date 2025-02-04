package links

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/jacekdobrowolski/goshort/pkg/logging"
	"github.com/jacekdobrowolski/goshort/pkg/telemetry"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewServer(logger *slog.Logger, pgStore *PostgresStore) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, logger, pgStore)

	var handler http.Handler = mux
	handler = logging.Middleware(handler, logger)

	return handler
}

func Run(ctx context.Context, w io.Writer, env func(string) string) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	resource, err := telemetry.NewResource(ctx, "goshort")
	if err != nil {
		return fmt.Errorf("error creating new resource: %w", err)
	}

	telemetryConn, err := grpc.NewClient(
		"collector.telemetry.svc.cluster.local:4317",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("error creating grpc connection %w", err)
	}

	logger, err := initLogger(ctx, resource, telemetryConn)
	if err != nil {
		logger := slog.New(
			slog.NewJSONHandler(
				w,
				&slog.HandlerOptions{
					AddSource: false,
					Level:     slog.LevelDebug,
				},
			),
		)

		logger.With(slog.String("application", "links"))
		logger.Error("logger init error", slog.String("err", err.Error()))
	}

	logger.Info("logger initialized")

	err = initTracer(ctx, resource, telemetryConn)
	if err != nil {
		logger.Error("tracing init error", slog.String("err", err.Error()))
	}

	requireEnv := func(variableName string) string {
		variable := env(variableName)
		if len(variable) == 0 {
			logger.Error("required Environment variable is empty or does not exist", "variable_name", variableName)
		}

		return variable
	}

	connectionString := fmt.Sprintf(
		"user=%s password=%s dbname=%s sslmode=disable host=%s port=%s",
		requireEnv("LINKS_POSTGRES_USER"),
		requireEnv("LINKS_POSTGRES_PASSWORD"),
		requireEnv("LINKS_POSTGRES_DBNAME"),
		requireEnv("LINKS_POSTGRES_HOST"),
		requireEnv("LINKS_POSTGRES_PORT"))

	pgStore, err := NewPostgresStore(ctx, connectionString, logger)
	if err != nil {
		return err
	}

	srv := NewServer(logger, pgStore)

	//nolint: mnd
	httpServer := &http.Server{
		Addr:         net.JoinHostPort("", "3000"),
		Handler:      srv,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
		}

		logger.Info("listening", "address", httpServer.Addr)
	}()

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "error shutting down http server: %s\n", err)
		}
	}()
	wg.Wait()

	return nil
}

func initLogger(
	ctx context.Context,
	resource *resource.Resource,
	telemetryConn *grpc.ClientConn,
) (*slog.Logger, error) {
	logExporter, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(telemetryConn))
	if err != nil {
		return nil, fmt.Errorf("error creating new otlp log exporter: %w", err)
	}

	logProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(resource),
		sdklog.WithProcessor(
			sdklog.NewSimpleProcessor(logExporter)),
	)

	logger := otelslog.NewLogger("links", otelslog.WithLoggerProvider(logProvider))

	return logger, nil
}

func initTracer(ctx context.Context, resource *resource.Resource, conn *grpc.ClientConn) error {
	traceExporter, err := telemetry.NewTraceExporter(ctx, conn)
	if err != nil {
		return fmt.Errorf("error creating new trace exporter %w", err)
	}

	simpleSpanProcessor := sdktrace.NewSimpleSpanProcessor(traceExporter)

	traceProvider := telemetry.NewTraceProvider(resource, simpleSpanProcessor)

	otel.SetTracerProvider(traceProvider)

	return nil
}
