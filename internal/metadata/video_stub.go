//go:build !cgo || !ffmpeg

package metadata

import (
	img "github.com/h2non/imaginary/internal/image"
)

// ExtractAV returns an error when FFmpeg libraries are not available.
// Build with `-tags cgo,ffmpeg` to enable audio and video metadata extraction.
func ExtractAV(buf []byte) (MetadataResult, error) {
	return MetadataResult{}, img.New(img.KindNotImplemented, "Audio and video metadata extraction requires FFmpeg (build with -tags cgo,ffmpeg)")
}
