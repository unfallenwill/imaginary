package image

import (
	"encoding/json"
	"strings"
)

// Kind represents the semantic category of an error in the image domain.
// It is independent of HTTP status codes — the server layer maps kinds to HTTP statuses.
type Kind int

const (
	KindUnknown         Kind = iota
	KindNotFound             // resource not found
	KindUnauthorized         // authentication failure
	KindForbidden            // authorization or signature mismatch
	KindMethodNotAllowed     // wrong HTTP method
	KindUnsupportedMedia     // image format not supported
	KindInvalidParam         // missing or malformed parameter
	KindEmptyBody            // empty or unreadable image data
	KindResolutionTooBig     // image resolution exceeds limit
	KindNotImplemented       // endpoint disabled
	KindProcessing           // error during image processing
)

// Error represents a domain-level image processing error.
// It carries a semantic Kind, not an HTTP status code.
type Error struct {
	Message string `json:"message,omitempty"`
	Kind    Kind   `json:"code"`
}

func (e Error) Error() string {
	return e.Message
}

// JSON serializes the error for wire transport.
// Note: the "status" field contains the semantic Kind code, not an HTTP status.
// The server layer provides the HTTP status mapping where needed.
func (e Error) JSON() []byte {
	type wire struct {
		Message string `json:"message,omitempty"`
		Status  int    `json:"status"`
	}
	buf, _ := json.Marshal(wire{Message: e.Message, Status: int(e.Kind)})
	return buf
}

// NewError creates an image.Error with an explicit message and semantic kind.
func NewError(message string, kind Kind) Error {
	message = strings.ReplaceAll(message, "\n", "")
	return Error{Message: message, Kind: kind}
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
)
