package image

import (
	"fmt"
	"io"
	"os"
	"path"
	"testing"

	"github.com/h2non/bimg"
)

func TestImageOperations(t *testing.T) {
	cases := []struct {
		name    string
		op      Operation
		opts    ImageOptions
		expectW int
		expectH int
	}{
		{"Resize/BothDims", Resize, ImageOptions{Width: 300, Height: 300}, 300, 300},
		{"Resize/WidthOnly", Resize, ImageOptions{Width: 300}, 300, 404},
		{"Resize/NoCropFalse", Resize, ImageOptions{Width: 300, NoCrop: boolPtr(false)}, 300, 740},
		{"Resize/NoCropTrue", Resize, ImageOptions{Width: 300, NoCrop: boolPtr(true)}, 300, 404},
		{"Fit", Fit, ImageOptions{Width: 300, Height: 300}, 223, 300},
		{"AutoRotate", AutoRotate, ImageOptions{}, 550, 740},
	}

	buf, _ := io.ReadAll(readImageFile("imaginary.jpg"))

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			img, err := tc.op(buf, tc.opts)
			if err != nil {
				t.Fatalf("Cannot process image: %s", err)
			}
			if img.Mime != "image/jpeg" {
				t.Error("Invalid image MIME type")
			}
			if err := assertImageSize(img.Body, tc.expectW, tc.expectH); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestImagePipelineOperations(t *testing.T) {
	width, height := 300, 260

	operations := PipelineOperations{
		PipelineOperation{
			Name: "crop",
			Params: PipelineParams{
				Width:  &width,
				Height: &height,
			},
		},
		PipelineOperation{
			Name: "convert",
			Params: PipelineParams{
				Type: "webp",
			},
		},
	}

	opts := ImageOptions{Operations: operations}
	buf, _ := io.ReadAll(readImageFile("imaginary.jpg"))

	img, err := Pipeline(buf, opts)
	if err != nil {
		t.Errorf("Cannot process image: %s", err)
	}
	if img.Mime != "image/webp" {
		t.Error("Invalid image MIME type")
	}
	if assertImageSize(img.Body, width, height) != nil {
		t.Errorf("Invalid image size, expected: %dx%d", width, height)
	}
}

func TestCalculateDestinationFitDimension(t *testing.T) {
	cases := []struct {
		// Image
		imageWidth  int
		imageHeight int

		// User parameter
		optionWidth  int
		optionHeight int

		// Expect
		fitWidth  int
		fitHeight int
	}{

		// Leading Width
		{1280, 1000, 710, 9999, 710, 555},
		{1279, 1000, 710, 9999, 710, 555},
		{900, 500, 312, 312, 312, 173}, // rounding down
		{900, 500, 313, 313, 313, 174}, // rounding up

		// Leading height
		{1299, 2000, 710, 999, 649, 999},
		{1500, 2000, 710, 999, 710, 947},
	}

	for _, tc := range cases {
		fitWidth, fitHeight := calculateDestinationFitDimension(tc.imageWidth, tc.imageHeight, tc.optionWidth, tc.optionHeight)
		if fitWidth != tc.fitWidth || fitHeight != tc.fitHeight {
			t.Errorf(
				"Fit dimensions calculation failure\nExpected : %d/%d (width/height)\nActual   : %d/%d (width/height)\n%+v",
				tc.fitWidth, tc.fitHeight, fitWidth, fitHeight, tc,
			)
		}
	}
}

// readImageFile reads a test fixture from the project root testdata directory.
func readImageFile(file string) io.Reader {
	buf, _ := os.Open(path.Join("../../testdata", file))
	return buf
}

// assertImageSize checks that the image buffer has the expected dimensions.
func assertImageSize(buf []byte, width, height int) error {
	size, err := bimg.NewImage(buf).Size()
	if err != nil {
		return err
	}
	if size.Width != width || size.Height != height {
		return fmt.Errorf("invalid image size: %dx%d, expected: %dx%d", size.Width, size.Height, width, height)
	}
	return nil
}
