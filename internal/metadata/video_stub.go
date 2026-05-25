//go:build !cgo || !ffmpeg

package metadata

import (
	img "github.com/h2non/imaginary/internal/image"
)

// ExtractVideo returns an error when FFmpeg libraries are not available.
// Build with `-tags cgo,ffmpeg` to enable video metadata extraction.
func ExtractVideo(buf []byte) (MetadataResult, error) {
	return MetadataResult{}, img.NewError("Video metadata extraction requires FFmpeg (build with -tags cgo,ffmpeg)", img.KindNotImplemented)
}
