package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/h2non/bimg"

	img "github.com/h2non/imaginary/internal/image"
)

// ErrorConfig holds error response behavior configuration.
type ErrorConfig struct {
	PlaceholderEnabled bool
	PlaceholderImage   []byte
	PlaceholderStatus  int
}

// kindToHTTP maps semantic error kinds to HTTP status codes.
// This is the single place where domain semantics become transport details.
var kindToHTTP = map[img.Kind]int{
	img.KindUnknown:          http.StatusInternalServerError,
	img.KindNotFound:         http.StatusNotFound,
	img.KindUnauthorized:     http.StatusUnauthorized,
	img.KindForbidden:        http.StatusForbidden,
	img.KindMethodNotAllowed: http.StatusMethodNotAllowed,
	img.KindUnsupportedMedia: http.StatusNotAcceptable,
	img.KindInvalidParam:     http.StatusBadRequest,
	img.KindEmptyBody:        http.StatusBadRequest,
	img.KindResolutionTooBig: http.StatusUnprocessableEntity,
	img.KindNotImplemented:   http.StatusNotImplemented,
	img.KindProcessing:       http.StatusBadRequest,
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
	bimgOptions := bimg.Options{
		Force:   true,
		Crop:    true,
		Enlarge: true,
		Type:    img.ImageType(r.URL.Query().Get("type")),
	}

	width, werr := strconv.Atoi(r.URL.Query().Get("width"))
	if werr != nil {
		sendErrorResponse(w, http.StatusBadRequest, werr)
		return
	}
	bimgOptions.Width = width

	height, herr := strconv.Atoi(r.URL.Query().Get("height"))
	if herr != nil {
		sendErrorResponse(w, http.StatusBadRequest, herr)
		return
	}
	bimgOptions.Height = height

	buf, err := bimg.Resize(cfg.PlaceholderImage, bimgOptions)
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	w.Header().Set("Content-Type", img.GetImageMimeType(bimg.DetermineImageType(buf)))
	status := httpStatusFor(errCaller)
	w.Header().Set("Error", string(errorReplyJSON(errCaller, status)))
	if cfg.PlaceholderStatus != 0 {
		w.WriteHeader(cfg.PlaceholderStatus)
	} else {
		w.WriteHeader(httpStatusFor(errCaller))
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
