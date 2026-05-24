package image

import (
	"encoding/json"
	"strings"
)

// Kind represents the semantic category of an error in the image domain.
// It is independent of HTTP status codes — the server layer maps kinds to HTTP statuses.
type Kind int

const (
	KindUnknown          Kind = iota
	KindNotFound              // resource not found
	KindUnauthorized          // authentication failure
	KindForbidden             // authorization or signature mismatch
	KindMethodNotAllowed      // wrong HTTP method
	KindUnsupportedMedia      // image format not supported
	KindInvalidParam          // missing or malformed parameter
	KindEmptyBody             // empty or unreadable image data
	KindResolutionTooBig      // image resolution exceeds limit
	KindNotImplemented        // endpoint disabled
	KindProcessing            // error during image processing
	KindUpstream              // upstream service (remote HTTP, object storage) error
)

// Error represents a domain-level image processing error.
// It carries a semantic Kind, not an HTTP status code.
// Error supports the Go 1.13+ error chain via Unwrap and Is,
// so callers can use errors.Is and errors.As to inspect it.
type Error struct {
	Message string `json:"message,omitempty"`
	Kind    Kind   `json:"code"`
	err     error
}

func (e *Error) Error() string {
	if e.err != nil {
		return e.Message + ": " + e.err.Error()
	}
	return e.Message
}

// Unwrap returns the wrapped underlying error, supporting errors.Is/As.
func (e *Error) Unwrap() error {
	return e.err
}

// Is supports errors.Is comparisons against sentinel Error values.
// Two *Error values match if they have the same Kind.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Kind == t.Kind
}

// JSON serializes the error for wire transport.
// Note: the "status" field contains the semantic Kind code, not an HTTP status.
// The server layer provides the HTTP status mapping where needed.
func (e *Error) JSON() []byte {
	type wire struct {
		Message string `json:"message,omitempty"`
		Status  int    `json:"status"`
	}
	buf, _ := json.Marshal(wire{Message: e.Message, Status: int(e.Kind)})
	return buf
}

// NewError creates an *image.Error with an explicit message and semantic kind.
func NewError(message string, kind Kind) *Error {
	message = strings.ReplaceAll(message, "\n", "")
	return &Error{Message: message, Kind: kind}
}

// WrapError creates an *image.Error that wraps an underlying error,
// preserving the error chain for errors.Is/As.
func WrapError(message string, kind Kind, err error) *Error {
	message = strings.ReplaceAll(message, "\n", "")
	return &Error{Message: message, Kind: kind, err: err}
}

var (
	ErrNotFound             = NewError("Not found", KindNotFound)
	ErrInvalidAPIKey        = NewError("Invalid or missing API key", KindUnauthorized)
	ErrMethodNotAllowed     = NewError("HTTP method not allowed. Try with a POST or GET method (-enable-url-source flag must be defined)", KindMethodNotAllowed)
	ErrGetMethodNotAllowed  = NewError("GET method not allowed. Make sure remote URL source is enabled by using the flag: -enable-url-source", KindMethodNotAllowed)
	ErrUnsupportedMedia     = NewError("Unsupported media type", KindUnsupportedMedia)
	ErrOutputFormat         = NewError("Unsupported output image format", KindInvalidParam)
	ErrEmptyBody            = NewError("Empty or unreadable image", KindEmptyBody)
	ErrMissingParamFile     = NewError("Missing required param: file", KindInvalidParam)
	ErrInvalidFilePath      = NewError("Invalid file path", KindInvalidParam)
	ErrInvalidImageURL      = NewError("Invalid image URL", KindInvalidParam)
	ErrMissingImageSource   = NewError("Cannot process the image due to missing or invalid params", KindInvalidParam)
	ErrNotImplemented       = NewError("Not implemented endpoint", KindNotImplemented)
	ErrInvalidURLSignature  = NewError("Invalid URL signature", KindInvalidParam)
	ErrURLSignatureMismatch = NewError("URL signature mismatch", KindForbidden)
	ErrResolutionTooBig     = NewError("Image resolution is too big", KindResolutionTooBig)
	ErrUpstream             = NewError("Upstream service error", KindUpstream)
	ErrOriginNotAllowed     = NewError("Remote URL origin not allowed", KindUpstream)
	ErrContentTooLarge      = NewError("Content size exceeds maximum allowed", KindInvalidParam)
)
