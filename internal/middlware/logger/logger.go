package logger

import (
	"bytes"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type responseWriter struct {
	http.ResponseWriter
	body   *bytes.Buffer
	size   int
	status int
}

func (rw *responseWriter) Size() int {
	return rw.size
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	rw.body.Write(b)
	return n, err
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Status() int {
	if rw.status == 0 {
		return http.StatusOK
	}
	return rw.status
}

func Logger(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{ResponseWriter: w, body: &bytes.Buffer{}}

			cookieHeader := r.Header.Get("Cookie")

			logger.Debug("Incoming request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("cookie", cookieHeader),
			)

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			var cookies []string
			cookies = append(cookies, rw.Header()["Set-Cookie"]...)

			logger.Debug("Outgoing response",
				zap.Int("status", rw.Status()),
				zap.Duration("duration", duration),
				zap.Int("response_size", rw.Size()),
				zap.String("response_body", rw.body.String()),
				zap.Strings("cookies", cookies),
			)

		})
	}
}
