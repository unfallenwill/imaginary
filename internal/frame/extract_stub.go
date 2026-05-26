//go:build !cgo || !ffmpeg

package frame

import (
	img "github.com/h2non/imaginary/internal/image"
)

// FrameResult holds the extracted video frame as a JPEG image.
type FrameResult struct {
	Body []byte
	Mime string
}

// Extract returns an error when FFmpeg libraries are not available.
// Build with `-tags cgo,ffmpeg` to enable video frame extraction.
func Extract(buf []byte, timeSeconds float64) (FrameResult, error) {
	return FrameResult{}, img.NewNotImplementedError(
		"Video frame extraction requires FFmpeg (build with -tags cgo,ffmpeg)",
	)
}
