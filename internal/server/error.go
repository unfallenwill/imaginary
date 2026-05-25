package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	img "github.com/h2non/imaginary/internal/image"
)

// Transport-layer error kinds for HTTP-specific error categories.
// Defined here (not in image package) because they represent transport
// concerns (auth, method routing), not image domain concepts.
const (
	kindUnauthorized     img.Kind = 100
	kindForbidden        img.Kind = 101
	kindMethodNotAllowed img.Kind = 102
)

// Transport-layer sentinel errors. These represent HTTP-level failures
// (bad API key, wrong method, invalid signature) that never originate
// from the image domain layer.
var (
	errInvalidAPIKey        = img.NewError("Invalid or missing API key", kindUnauthorized)
	errMethodNotAllowed     = img.NewError("HTTP method not allowed. Try with a POST or GET method (-enable-url-source flag must be defined)", kindMethodNotAllowed)
	errGetMethodNotAllowed  = img.NewError("GET method not allowed. Make sure remote URL source is enabled by using the flag: -enable-url-source", kindMethodNotAllowed)
	errInvalidURLSignature  = img.NewError("Invalid URL signature", img.KindInvalidParam)
	errURLSignatureMismatch = img.NewError("URL signature mismatch", kindForbidden)
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

// kindToHTTP maps semantic error kinds to HTTP status codes.
// This is the single place where domain semantics become transport details.
var kindToHTTP = map[img.Kind]int{
	img.KindUnknown:          http.StatusInternalServerError,
	img.KindNotFound:         http.StatusNotFound,
	kindUnauthorized:         http.StatusUnauthorized,
	kindForbidden:            http.StatusForbidden,
	kindMethodNotAllowed:     http.StatusMethodNotAllowed,
	img.KindUnsupportedMedia: http.StatusNotAcceptable,
	img.KindInvalidParam:     http.StatusBadRequest,
	img.KindEmptyBody:        http.StatusBadRequest,
	img.KindResolutionTooBig: http.StatusUnprocessableEntity,
	img.KindNotImplemented:   http.StatusNotImplemented,
	img.KindProcessing:       http.StatusInternalServerError,
	img.KindUpstream:         http.StatusBadGateway,
}

// httpStatusFor maps an *image.Error's Kind to an HTTP status code.
func httpStatusFor(err *img.Error) int {
	if code, ok := kindToHTTP[err.Kind]; ok {
		return code
	}
	return http.StatusInternalServerError
}

type errorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

// ErrorReply writes an *image.Error as an HTTP JSON response.
// If placeholder is configured, it renders the placeholder image instead.
func ErrorReply(w http.ResponseWriter, r *http.Request, err *img.Error, cfg ErrorConfig) {
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

func replyWithPlaceholder(w http.ResponseWriter, r *http.Request, errCaller *img.Error, cfg ErrorConfig) {
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

	buf, mime, err := cfg.Resizer.ResizePlaceholder(cfg.PlaceholderImage, width, height, imageType)
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, err)
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
	_, _ = w.Write(buf)
}

func sendErrorResponse(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	buf, _ := json.Marshal(errorResponse{Error: err.Error(), Status: statusCode})
	_, _ = w.Write(buf)
}

func errorReplyJSON(err *img.Error, httpStatus int) []byte {
	buf, _ := json.Marshal(errorResponse{Error: err.Error(), Status: httpStatus})
	return buf
}
