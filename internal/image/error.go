package image

import (
	"strings"
)

// Error is the base error type for the image domain.
// It carries a human-readable message and supports Go error chains via Unwrap.
// Domain-specific error types contain *Error to classify themselves for the
// server layer, which uses errors.As to determine HTTP status codes.
type Error struct {
	msg string
	err error
}

func (e *Error) Error() string {
	if e.err != nil {
		return e.msg + ": " + e.err.Error()
	}
	return e.msg
}

// Unwrap returns the wrapped underlying error, supporting errors.Is/As.
func (e *Error) Unwrap() error {
	return e.err
}

// NewError creates a base domain error with the given message.
// Used by both domain-level typed constructors and the server transport layer
// (which wraps *Error into transport-specific error types).
func NewError(message string) *Error {
	return &Error{msg: strings.ReplaceAll(message, "\n", "")}
}

// WrapError creates a base domain error wrapping an underlying error.
func WrapError(message string, err error) *Error {
	return &Error{msg: strings.ReplaceAll(message, "\n", ""), err: err}
}

// --- Domain error types ---
//
// Each type represents a semantic error category in the image domain.
// The server layer uses errors.As to match these types and map them to
// appropriate HTTP status codes. This keeps the domain layer completely
// decoupled from HTTP concerns.
//
// Each type stores a *Error in a named field (not embedded) to avoid
// the field/method name collision between the embedded "*Error" field
// and the promoted Error() method from *image.Error.

// InvalidParamError indicates a missing or malformed parameter.
type InvalidParamError struct{ Err *Error }

func (e *InvalidParamError) Error() string { return e.Err.Error() }
func (e *InvalidParamError) Unwrap() error { return e.Err.err }

func NewInvalidParamError(message string) *InvalidParamError {
	return &InvalidParamError{Err: NewError(message)}
}

func WrapInvalidParamError(message string, err error) *InvalidParamError {
	return &InvalidParamError{Err: WrapError(message, err)}
}

// UnsupportedMediaError indicates the image format is not supported.
type UnsupportedMediaError struct{ Err *Error }

func (e *UnsupportedMediaError) Error() string { return e.Err.Error() }
func (e *UnsupportedMediaError) Unwrap() error { return e.Err.err }

func NewUnsupportedMediaError(message string) *UnsupportedMediaError {
	return &UnsupportedMediaError{Err: NewError(message)}
}

func WrapUnsupportedMediaError(message string, err error) *UnsupportedMediaError {
	return &UnsupportedMediaError{Err: WrapError(message, err)}
}

// EmptyBodyError indicates empty or unreadable image data.
type EmptyBodyError struct{ Err *Error }

func (e *EmptyBodyError) Error() string { return e.Err.Error() }
func (e *EmptyBodyError) Unwrap() error { return e.Err.err }

func NewEmptyBodyError(message string) *EmptyBodyError {
	return &EmptyBodyError{Err: NewError(message)}
}

// NotFoundError indicates a resource was not found.
type NotFoundError struct{ Err *Error }

func (e *NotFoundError) Error() string { return e.Err.Error() }
func (e *NotFoundError) Unwrap() error { return e.Err.err }

func NewNotFoundError(message string) *NotFoundError {
	return &NotFoundError{Err: NewError(message)}
}

func WrapNotFoundError(message string, err error) *NotFoundError {
	return &NotFoundError{Err: WrapError(message, err)}
}

// ResolutionTooBigError indicates the image resolution exceeds the limit.
type ResolutionTooBigError struct{ Err *Error }

func (e *ResolutionTooBigError) Error() string { return e.Err.Error() }
func (e *ResolutionTooBigError) Unwrap() error { return e.Err.err }

func NewResolutionTooBigError(message string) *ResolutionTooBigError {
	return &ResolutionTooBigError{Err: NewError(message)}
}

// NotImplementedError indicates the endpoint or feature is disabled.
type NotImplementedError struct{ Err *Error }

func (e *NotImplementedError) Error() string { return e.Err.Error() }
func (e *NotImplementedError) Unwrap() error { return e.Err.err }

func NewNotImplementedError(message string) *NotImplementedError {
	return &NotImplementedError{Err: NewError(message)}
}

// ProcessingError indicates an error during image processing.
type ProcessingError struct{ Err *Error }

func (e *ProcessingError) Error() string { return e.Err.Error() }
func (e *ProcessingError) Unwrap() error { return e.Err.err }

func NewProcessingError(message string) *ProcessingError {
	return &ProcessingError{Err: NewError(message)}
}

func WrapProcessingError(message string, err error) *ProcessingError {
	return &ProcessingError{Err: WrapError(message, err)}
}

// UpstreamError indicates a failure from an upstream service.
type UpstreamError struct{ Err *Error }

func (e *UpstreamError) Error() string { return e.Err.Error() }
func (e *UpstreamError) Unwrap() error { return e.Err.err }

func NewUpstreamError(message string) *UpstreamError {
	return &UpstreamError{Err: NewError(message)}
}

func WrapUpstreamError(message string, err error) *UpstreamError {
	return &UpstreamError{Err: WrapError(message, err)}
}

// --- Sentinel errors ---

var (
	ErrNotFound           = &NotFoundError{Err: NewError("Not found")}
	ErrUnsupportedMedia   = &UnsupportedMediaError{Err: NewError("Unsupported media type")}
	ErrOutputFormat       = &InvalidParamError{Err: NewError("Unsupported output image format")}
	ErrEmptyBody          = &EmptyBodyError{Err: NewError("Empty or unreadable image")}
	ErrMissingParamFile   = &InvalidParamError{Err: NewError("Missing required param: file")}
	ErrInvalidFilePath    = &InvalidParamError{Err: NewError("Invalid file path")}
	ErrInvalidImageURL    = &InvalidParamError{Err: NewError("Invalid image URL")}
	ErrMissingImageSource = &InvalidParamError{Err: NewError("Cannot process the image due to missing or invalid params")}
	ErrNotImplemented     = &NotImplementedError{Err: NewError("Not implemented endpoint")}
	ErrResolutionTooBig   = &ResolutionTooBigError{Err: NewError("Image resolution is too big")}
	ErrUpstream           = &UpstreamError{Err: NewError("Upstream service error")}
	ErrOriginNotAllowed   = &UpstreamError{Err: NewError("Remote URL origin not allowed")}
	ErrContentTooLarge    = &InvalidParamError{Err: NewError("Content size exceeds maximum allowed")}
)

