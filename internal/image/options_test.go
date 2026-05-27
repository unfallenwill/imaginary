package image_test

import (
	"testing"

	"github.com/h2non/imaginary/internal/image"
)

func TestBimgOptions(t *testing.T) {
	imgOpts := image.ImageOptions{
		Dimensions: image.Dimensions{Width: 500, Height: 600},
	}
	opts := image.BimgOptions(imgOpts)

	if opts.Width != imgOpts.Width || opts.Height != imgOpts.Height {
		t.Error("Invalid width and height")
	}
}
