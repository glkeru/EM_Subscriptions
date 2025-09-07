package emsub

import (
	"bytes"
	"io"
	"net/http"
	"time"

	config "github.com/glkeru/EM_Subscriptions/internal/config"
	"go.uber.org/zap"
)

const MaxBody = 1024

// логируем вызовы
type logResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *logResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
func MiddlewareLog(logger *zap.Logger, c *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			reqtime := time.Now()
			rid := r.Header.Get("X-Request-ID")

			// логируем тело запроса, если включено
			var logbody []byte
			if c.LogBody {
				savebody, err := io.ReadAll(r.Body)
				if err == nil {
					_ = r.Body.Close()
					if len(savebody) > MaxBody {
						logbody = savebody[:MaxBody]
					} else {
						logbody = savebody
					}
					r.Body = io.NopCloser(bytes.NewReader(logbody))
				}
			}

			logrw := &logResponseWriter{w, 200}
			next.ServeHTTP(logrw, r)

			logger.Info("http_request",
				zap.String("rid", rid),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("query", r.URL.RawQuery),
				zap.Int("status", logrw.status),
				zap.Duration("dur", time.Since(reqtime)),
				zap.String("ip", r.RemoteAddr),
				zap.String("ua", r.UserAgent()),
				zap.String("body", string(logbody)),
			)
		})
	}
}
