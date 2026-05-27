package server

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// LogRecord implements an HTTP logging record that captures request/response metadata.
type LogRecord struct {
	http.ResponseWriter
	status        int
	responseBytes int64
	ip            string
	method        string
	uri           string
	protocol      string
}

// Write acts like a proxy passing the given bytes buffer to the ResponseWriter
// and additionally counting the passed amount of bytes for logging usage.
func (r *LogRecord) Write(p []byte) (int, error) {
	written, err := r.ResponseWriter.Write(p)
	r.responseBytes += int64(written)
	return written, err
}

// WriteHeader calls ResponseWriter.WriteHeader() and sets the status code.
func (r *LogRecord) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// LogHandler maps the HTTP handler with a custom io.Writer compatible stream.
type LogHandler struct {
	handler http.Handler
	logger  *slog.Logger
	level   slog.Level
}

// NewLog creates a new logger using slog with JSON output.
func NewLog(handler http.Handler, out io.Writer, logLevel string) http.Handler {
	level := parseLogLevel(logLevel)
	handlerOpts := &slog.HandlerOptions{
		Level: level,
	}
	logger := slog.New(slog.NewJSONHandler(out, handlerOpts))
	return &LogHandler{handler: handler, logger: logger, level: level}
}

func parseLogLevel(logLevel string) slog.Level {
	switch strings.ToLower(logLevel) {
	case "error":
		return slog.LevelError
	case "warning", "warn":
		return slog.LevelWarn
	case "info":
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}

// ServeHTTP implements the standard HTTP handler, serving the request and logging structured output.
func (h *LogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	if colon := strings.LastIndex(clientIP, ":"); colon != -1 {
		clientIP = clientIP[:colon]
	}

	record := &LogRecord{
		ResponseWriter: w,
		ip:             clientIP,
		method:         r.Method,
		uri:            r.RequestURI,
		protocol:       r.Proto,
		status:         http.StatusOK,
	}

	startTime := time.Now()
	h.handler.ServeHTTP(record, r)
	elapsedTime := time.Since(startTime)

	// Determine the appropriate log level based on status code.
	var level slog.Level
	switch {
	case record.status >= http.StatusInternalServerError:
		level = slog.LevelError
	case record.status >= http.StatusBadRequest:
		level = slog.LevelWarn
	default:
		level = slog.LevelInfo
	}

	// Skip logging if the determined level is below the configured threshold.
	if !h.logger.Enabled(r.Context(), level) {
		return
	}

	h.logger.LogAttrs(r.Context(), level, "http request",
		slog.String("method", record.method),
		slog.String("path", record.uri),
		slog.Int("status", record.status),
		slog.Duration("duration", elapsedTime),
		slog.Int64("bytes", record.responseBytes),
		slog.String("ip", record.ip),
	)
}
