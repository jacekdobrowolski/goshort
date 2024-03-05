package logging

import (
	"log/slog"
	"net/http"
	"time"
)

func Middleware(h http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("request received", "method", r.Method, "url", r.URL)
		start := time.Now()
		h.ServeHTTP(w, r)
		logger.Info("response written", "time", time.Since(start))
	})
}
