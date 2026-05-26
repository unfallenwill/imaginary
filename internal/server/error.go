package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	img "github.com/h2non/imaginary/internal/image"
)

// Transport-layer error types. These represent HTTP-level failures
// (bad API key, wrong method, invalid signature) that never originate
// from the image domain layer.

// UnauthorizedError indicates authentication failure (HTTP 401).
type UnauthorizedError struct{ Err *img.Error }

func (e *UnauthorizedError) Error() string { return e.Err.Error() }
func (e *UnauthorizedError) Unwrap() error { return e.Err.Unwrap() }

// ForbiddenError indicates authorization failure (HTTP 403).
type ForbiddenError struct{ Err *img.Error }

func (e *ForbiddenError) Error() string { return e.Err.Error() }
func (e *ForbiddenError) Unwrap() error { return e.Err.Unwrap() }

// MethodNotAllowedError indicates the HTTP method is not supported (HTTP 405).
type MethodNotAllowedError struct{ Err *img.Error }

func (e *MethodNotAllowedError) Error() string { return e.Err.Error() }
func (e *MethodNotAllowedError) Unwrap() error { return e.Err.Unwrap() }

var (
	errInvalidAPIKey        = &UnauthorizedError{Err: img.NewError("Invalid or missing API key")}
	errMethodNotAllowed     = &MethodNotAllowedError{Err: img.NewError("HTTP method not allowed. Try with a POST or GET method (-enable-url-source flag must be defined)")}
	errGetMethodNotAllowed  = &MethodNotAllowedError{Err: img.NewError("GET method not allowed. Make sure remote URL source is enabled by using the flag: -enable-url-source")}
	errInvalidURLSignature  = &img.InvalidParamError{Err: img.NewError("Invalid URL signature")}
	errURLSignatureMismatch = &ForbiddenError{Err: img.NewError("URL signature mismatch")}
)

// PlaceholderResizer abstracts the image resize operation used to render
// placeholder error responses. Defined at the consumer so the error layer
// doesn't directly depend on bimg/libvips.
type PlaceholderResizer interface {
	ResizePlaceholder(buf []byte, width, height int, imageType string) ([]byte, string, error)
}

// ErrorConfig holds error response behavior configuration.
type ErrorConfig struct {
	PlaceholderEnabled bool
	PlaceholderImage   []byte
	PlaceholderStatus  int
	Resizer            PlaceholderResizer
}

// httpStatusFor maps an error to an HTTP status code using Go's type system.
// This is the single place where domain/transport error types become HTTP details.
func httpStatusFor(err error) int {
	// Transport-layer errors (check first — most specific)
	var ue *UnauthorizedError
	if errors.As(err, &ue) {
		return http.StatusUnauthorized
	}
	var fe *ForbiddenError
	if errors.As(err, &fe) {
		return http.StatusForbidden
	}
	var mne *MethodNotAllowedError
	if errors.As(err, &mne) {
		return http.StatusMethodNotAllowed
	}

	// Domain-layer errors
	var nfe *img.NotFoundError
	if errors.As(err, &nfe) {
		return http.StatusNotFound
	}
	var ume *img.UnsupportedMediaError
	if errors.As(err, &ume) {
		return http.StatusNotAcceptable
	}
	var ipe *img.InvalidParamError
	if errors.As(err, &ipe) {
		return http.StatusBadRequest
	}
	var ebe *img.EmptyBodyError
	if errors.As(err, &ebe) {
		return http.StatusBadRequest
	}
	var rtbe *img.ResolutionTooBigError
	if errors.As(err, &rtbe) {
		return http.StatusUnprocessableEntity
	}
	var nie *img.NotImplementedError
	if errors.As(err, &nie) {
		return http.StatusNotImplemented
	}
	var pe *img.ProcessingError
	if errors.As(err, &pe) {
		return http.StatusInternalServerError
	}
	var use *img.UpstreamError
	if errors.As(err, &use) {
		return http.StatusBadGateway
	}

	return http.StatusInternalServerError
}

type errorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

// ErrorReply writes an error as an HTTP JSON response.
// If placeholder is configured, it renders the placeholder image instead.
func ErrorReply(w http.ResponseWriter, r *http.Request, err error, cfg ErrorConfig) {
	if cfg.PlaceholderEnabled {
		replyWithPlaceholder(w, r, err, cfg)
		return
	}

	status := httpStatusFor(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	buf, _ := json.Marshal(errorResponse{Error: err.Error(), Status: status})
	_, _ = w.Write(buf)
}

func replyWithPlaceholder(w http.ResponseWriter, r *http.Request, errCaller error, cfg ErrorConfig) {
	if cfg.Resizer == nil {
		// No resizer configured — fall back to JSON error response.
		status := httpStatusFor(errCaller)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		buf, _ := json.Marshal(errorResponse{Error: errCaller.Error(), Status: status})
		_, _ = w.Write(buf)
		return
	}

	width, werr := strconv.Atoi(r.URL.Query().Get("width"))
	if werr != nil {
		sendErrorResponse(w, http.StatusBadRequest, werr)
		return
	}

	height, herr := strconv.Atoi(r.URL.Query().Get("height"))
	if herr != nil {
		sendErrorResponse(w, http.StatusBadRequest, herr)
		return
	}

	imageType := r.URL.Query().Get("type")

	buf, mime, rerr := cfg.Resizer.ResizePlaceholder(cfg.PlaceholderImage, width, height, imageType)
	if rerr != nil {
		sendErrorResponse(w, http.StatusBadRequest, rerr)
		return
	}

	w.Header().Set("Content-Type", mime)
	status := httpStatusFor(errCaller)
	w.Header().Set("Error", string(errorReplyJSON(errCaller, status)))
	if cfg.PlaceholderStatus != 0 {
		w.WriteHeader(cfg.PlaceholderStatus)
	} else {
		w.WriteHeader(status)
	}
	_, _ = w.Write(buf) // #nosec G705 -- buf is a placeholder image (not user-controlled HTML)
}

func sendErrorResponse(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	buf, _ := json.Marshal(errorResponse{Error: err.Error(), Status: statusCode})
	_, _ = w.Write(buf)
}

func errorReplyJSON(err error, httpStatus int) []byte {
	buf, _ := json.Marshal(errorResponse{Error: err.Error(), Status: httpStatus})
	return buf
}
