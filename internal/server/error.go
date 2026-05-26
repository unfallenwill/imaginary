package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	img "github.com/h2non/imaginary/internal/image"
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

// --- Sentinel errors (transport-level) ---

var (
	errInvalidAPIKey        = img.New(img.KindUnauthorized, "Invalid or missing API key")
	errMethodNotAllowed     = img.New(img.KindMethodNotAllowed, "HTTP method not allowed. Try with a POST or GET method (-enable-url-source flag must be defined)")
	errGetMethodNotAllowed  = img.New(img.KindMethodNotAllowed, "GET method not allowed. Make sure remote URL source is enabled by using the flag: -enable-url-source")
	errInvalidURLSignature  = img.New(img.KindInvalidParam, "Invalid URL signature")
	errURLSignatureMismatch = img.New(img.KindForbidden, "URL signature mismatch")
)

// httpStatusFor maps an error to an HTTP status code.
// It extracts the *image.Error Kind — the single place where domain errors
// become HTTP details.
func httpStatusFor(err error) int {
	var imgErr *img.Error
	if errors.As(err, &imgErr) {
		switch imgErr.Kind {
		case img.KindInvalidParam, img.KindEmptyBody:
			return http.StatusBadRequest
		case img.KindNotFound:
			return http.StatusNotFound
		case img.KindUnsupportedMedia:
			return http.StatusNotAcceptable
		case img.KindResolutionTooBig:
			return http.StatusUnprocessableEntity
		case img.KindNotImplemented:
			return http.StatusNotImplemented
		case img.KindProcessing:
			return http.StatusInternalServerError
		case img.KindUpstream:
			return http.StatusBadGateway
		case img.KindUnauthorized:
			return http.StatusUnauthorized
		case img.KindForbidden:
			return http.StatusForbidden
		case img.KindMethodNotAllowed:
			return http.StatusMethodNotAllowed
		}
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
	_, _ = w.Write(buf) // #nosec G705 -- buf is a placeholder image, not user-controlled HTML
}

func sendErrorResponse(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	buf, _ := json.Marshal(errorResponse{Error: err.Error(), Status: statusCode})
	_, _ = w.Write(buf) // #nosec G705 -- buf is JSON, not user-controlled HTML
}

func errorReplyJSON(err error, httpStatus int) []byte {
	buf, _ := json.Marshal(errorResponse{Error: err.Error(), Status: httpStatus})
	return buf
}
