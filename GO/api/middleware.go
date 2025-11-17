package api

import (
	"net/http"
	"time"

	"github.com/Halturshik/EM-test-task/GO/logger"
)

type logResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *logResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lrw := &logResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		logger.Info("→ %s %s", r.Method, r.URL.Path)

		next.ServeHTTP(lrw, r)

		duration := time.Since(start)

		if lrw.statusCode >= 400 {
			logger.Error("← %s %s завершился с ошибкой %d (заняло %s)", r.Method, r.URL.Path, lrw.statusCode, duration)
		} else {
			logger.Info("← %s %s (заняло %s)", r.Method, r.URL.Path, duration)
		}
	})
}
