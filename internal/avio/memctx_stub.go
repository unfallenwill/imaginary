//go:build !cgo || !ffmpeg

package avio

import (
	img "github.com/h2non/imaginary/internal/image"
)

// NewMemoryIOContext returns an error when FFmpeg libraries are not available.
// Build with `-tags cgo,ffmpeg` to enable FFmpeg-based memory I/O.
func NewMemoryIOContext(data []byte) (struct{}, error) {
	return struct{}{}, img.New(img.KindNotImplemented, "FFmpeg memory I/O requires FFmpeg (build with -tags cgo,ffmpeg)")
}
