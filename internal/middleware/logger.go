package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mtsgn/mtsgn-mtsgn-system-common-svc/common/logger"
	"github.com/mtsgn/mtsgn-system-gateway-svc/pkg/utils"
)

// Logger logs HTTP requests with status code, method, and duration
func Logger(next http.Handler, logger logger.ZeroLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		start := time.Now()
		clientIP := utils.GetClientIP(r)
		requestID := uuid.New().String()

		// Log request details (except body for large requests)
		logger.Info(ctx,
			"Incoming request",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"client_ip", clientIP,
			"user_agent", r.UserAgent(),
		)

		// For debugging, log request body (be careful with large bodies)
		if r.Body != nil {
			bodyBytes, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			if len(bodyBytes) > 0 && len(bodyBytes) < 1024*10 { // Limit to 10KB
				logger.Debug(ctx, "Request body", "body", bodyBytes)
			}
		}

		// Use response recorder to capture status and duration
		recorder := newResponseRecorder(w)
		next.ServeHTTP(recorder, r)

		duration := time.Since(start)

		// Log access
		accessLog(logger, ctx, r, requestID, recorder.statusCode, duration, clientIP)

		// Log error responses
		if recorder.statusCode >= 400 {
			logger.Error(ctx, "Request failed", "status", recorder.statusCode, "path", r.URL.Path, "duration", duration, "client_ip", clientIP)
		}
	})
}

func accessLog(logger logger.ZeroLogger, ctx context.Context, r *http.Request, requestID string, status int, duration time.Duration, clientIP string) {
	logger.Info(ctx, "HTTP request", "request_id", requestID, "method", r.Method, "path", r.URL.Path, "status", status, "duration", duration, "client_ip", clientIP, "user_agent", r.UserAgent())
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           bytes.NewBuffer(nil),
	}
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}
