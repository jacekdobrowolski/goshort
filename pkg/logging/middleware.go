package logging

import (
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
)

func Middleware(handler http.Handler, logger *slog.Logger) http.Handler {
	tracer := otel.Tracer("links-tracer")

	return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		logger.Info("request received", "method", req.Method, "url", req.URL)

		ctx, parentSpan := tracer.Start(req.Context(), "http")
		defer parentSpan.End()

		req = req.WithContext(ctx)

		start := time.Now()

		handler.ServeHTTP(writer, req)

		logger.Info("response written", "time", time.Since(start))
	})
}
