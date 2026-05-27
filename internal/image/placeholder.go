package image

import (
	"encoding/base64"
	"io"
	"strings"
)

const placeholderData = `/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAYEBQYFBAYGBQYHBwYIChAKCgkJChQODwwQFxQYGBcUFhYaHSUfGhsjHBYWICwgIyYnKSopGR8tMC0oMCUoKSj/2wBDAQcHBwoIChMKChMoGhYaKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCj/wAARCAGQAZADASIAAhEBAxEB/8QAGwABAAMBAQEBAAAAAAAAAAAAAAQFBgMCAQf/xAA5EAEAAQQBAQUFBAcJAAAAAAAAAQIDBBEFEgYUITFRE0FzobEVNWHBFjZTcZGS0SIjNIGCg8Lh8f/EABQBAQAAAAAAAAAAAAAAAAAAAAD/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwD9lAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB//2Q==`

// Placeholder is the embedded fallback image used when -enable-placeholder is set
// without an explicit -placeholder path.
var Placeholder, _ = io.ReadAll(base64.NewDecoder(base64.StdEncoding, strings.NewReader(placeholderData)))

// BimgResizer resizes placeholder images using the image domain layer.
// It implements the PlaceholderResizer interface defined in the server package.
type BimgResizer struct{}

// ResizePlaceholder resizes the given image buffer to the specified dimensions
// and output type, returning the resized bytes and MIME type.
func (BimgResizer) ResizePlaceholder(buf []byte, width, height int, imageType string) ([]byte, string, error) {
	opts := BimgOptions(ImageOptions{
		Dimensions: Dimensions{Width: width, Height: height},
		Type:       imageType,
	})
	opts.Force = true
	opts.Crop = true
	opts.Enlarge = true

	result, err := Process(buf, opts)
	if err != nil {
		return nil, "", err
	}
	return result.Body, result.Mime, nil
}
