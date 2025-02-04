package logging

import (
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func Middleware(handler http.Handler, logger *slog.Logger) http.Handler {
	tracer := otel.Tracer("links-tracer")

	return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		ctx, parentSpan := tracer.Start(req.Context(), "http", trace.WithNewRoot())
		defer parentSpan.End()

		logger.Info("request received",
			"http.method", req.Method,
			"http.url", req.URL.Path,
			"trace_id", parentSpan.SpanContext().TraceID().String(),
		)

		parentSpan.SetAttributes(
			attribute.KeyValue{
				Key:   "url.full",
				Value: attribute.StringValue(req.URL.Path),
			},
			attribute.KeyValue{
				Key:   "http.request.mehtod",
				Value: attribute.StringValue(req.Method),
			},
		)

		req = req.WithContext(ctx)

		start := time.Now()

		handler.ServeHTTP(writer, req)

		logger.Info("response written", "time", time.Since(start))
	})
}
