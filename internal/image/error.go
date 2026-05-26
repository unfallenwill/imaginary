package image

import "strings"

// Kind classifies an error into a semantic category.
// The server layer maps each Kind to an HTTP status code.
type Kind uint8

const (
	KindInvalidParam     Kind = iota // malformed or missing parameter
	KindNotFound                     // resource not found
	KindUnsupportedMedia             // unsupported image format
	KindEmptyBody                    // empty or unreadable image data
	KindResolutionTooBig             // image exceeds megapixel limit
	KindNotImplemented               // endpoint or feature disabled/unsupported
	KindProcessing                   // libvips/ffmpeg processing failure
	KindUpstream                     // upstream service failure
	KindUnauthorized                 // missing or invalid credentials
	KindForbidden                    // valid credentials but insufficient permission
	KindMethodNotAllowed             // unsupported HTTP method
)

// Error is the single domain/transport error type.
// It carries a Kind for classification (mapped to HTTP status by the server),
// a human-readable message, and an optional underlying cause.
type Error struct {
	Kind    Kind
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap returns the underlying cause, supporting errors.Is and errors.As.
func (e *Error) Unwrap() error { return e.Cause }

// New creates a new Error of the given kind with a sanitized message.
func New(kind Kind, msg string) *Error {
	return &Error{Kind: kind, Message: strings.ReplaceAll(msg, "\n", "")}
}

// Wrap creates a new Error wrapping an underlying cause.
func Wrap(kind Kind, msg string, cause error) *Error {
	return &Error{Kind: kind, Message: strings.ReplaceAll(msg, "\n", ""), Cause: cause}
}

// --- Sentinel errors ---

var (
	ErrNotFound           = New(KindNotFound, "Not found")
	ErrUnsupportedMedia   = New(KindUnsupportedMedia, "Unsupported media type")
	ErrOutputFormat       = New(KindInvalidParam, "Unsupported output image format")
	ErrEmptyBody          = New(KindEmptyBody, "Empty or unreadable image")
	ErrMissingParamFile   = New(KindInvalidParam, "Missing required param: file")
	ErrInvalidFilePath    = New(KindInvalidParam, "Invalid file path")
	ErrInvalidImageURL    = New(KindInvalidParam, "Invalid image URL")
	ErrMissingImageSource = New(KindInvalidParam, "Cannot process the image due to missing or invalid params")
	ErrNotImplemented     = New(KindNotImplemented, "Not implemented endpoint")
	ErrResolutionTooBig   = New(KindResolutionTooBig, "Image resolution is too big")
	ErrUpstream           = New(KindUpstream, "Upstream service error")
	ErrOriginNotAllowed   = New(KindUpstream, "Remote URL origin not allowed")
	ErrContentTooLarge    = New(KindInvalidParam, "Content size exceeds maximum allowed")
)
