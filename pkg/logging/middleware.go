package logging

import (
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
)

func Middleware(h http.Handler, logger *slog.Logger) http.Handler {
	tracer := otel.Tracer("links-tracer")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("request received", "method", r.Method, "url", r.URL)

		ctx, parentSpan := tracer.Start(r.Context(), "http")
		defer parentSpan.End()

		r = r.WithContext(ctx)

		start := time.Now()

		h.ServeHTTP(w, r)

		logger.Info("response written", "time", time.Since(start))
	})
}
