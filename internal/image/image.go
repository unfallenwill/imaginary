package image

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/h2non/bimg"
)

// OperationsMap defines the allowed image transformation operations listed by name.
// Used for pipeline image processing.
var OperationsMap = map[string]Operation{
	"crop":           Crop,
	"resize":         Resize,
	"enlarge":        Enlarge,
	"extract":        Extract,
	"rotate":         Rotate,
	"autorotate":     AutoRotate,
	"flip":           Flip,
	"flop":           Flop,
	"thumbnail":      Thumbnail,
	"zoom":           Zoom,
	"convert":        Convert,
	"watermark":      Watermark,
	"watermarkImage": WatermarkImage,
	"blur":           GaussianBlur,
	"smartcrop":      SmartCrop,
	"fit":            Fit,
}

// Image stores an image binary buffer and its MIME type
type Image struct {
	Body []byte
	Mime string
}

// Operation implements an image transformation runnable interface
type Operation func([]byte, ImageOptions) (Image, error)

// Run performs the image transformation
func (o Operation) Run(buf []byte, opts ImageOptions) (Image, error) {
	return o(buf, opts)
}

// ImageInfo represents an image details and additional metadata
type ImageInfo struct {
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Type        string `json:"type"`
	Space       string `json:"space"`
	Alpha       bool   `json:"hasAlpha"`
	Profile     bool   `json:"hasProfile"`
	Channels    int    `json:"channels"`
	Orientation int    `json:"orientation"`
}

// Info returns image metadata as JSON.
func Info(buf []byte, o ImageOptions) (Image, error) {
	image := Image{Mime: "application/json"}

	meta, err := bimg.Metadata(buf)
	if err != nil {
		return image, WrapProcessingError("Cannot retrieve image metadata", err)
	}

	info := ImageInfo{
		Width:       meta.Size.Width,
		Height:      meta.Size.Height,
		Type:        meta.Type,
		Space:       meta.Space,
		Alpha:       meta.Alpha,
		Profile:     meta.Profile,
		Channels:    meta.Channels,
		Orientation: meta.Orientation,
	}

	body, _ := json.Marshal(info)
	image.Body = body

	return image, nil
}

// Resize resizes the image to the given width and height.
func Resize(buf []byte, o ImageOptions) (Image, error) {
	if o.Width == 0 && o.Height == 0 {
		return Image{}, NewInvalidParamError("Missing required param: height or width")
	}

	opts := BimgOptions(o)
	opts.Embed = true

	if o.NoCrop != nil {
		opts.Crop = !*o.NoCrop
	}

	return Process(buf, opts)
}

// Fit resizes the image to fit within the given dimensions while preserving aspect ratio.
func Fit(buf []byte, o ImageOptions) (Image, error) {
	if o.Width == 0 || o.Height == 0 {
		return Image{}, NewInvalidParamError("Missing required params: height, width")
	}

	metadata, err := bimg.Metadata(buf)
	if err != nil {
		return Image{}, WrapProcessingError("cannot retrieve image metadata", err)
	}

	dims := metadata.Size

	if dims.Width == 0 || dims.Height == 0 {
		return Image{}, NewUnsupportedMediaError("Width or height of requested image is zero")
	}

	var originHeight, originWidth int
	var fitHeight, fitWidth *int
	if derefBool(o.NoRotation, false) || (metadata.Orientation <= 4) {
		originHeight = dims.Height
		originWidth = dims.Width
		fitHeight = &o.Height
		fitWidth = &o.Width
	} else {
		originWidth = dims.Height
		originHeight = dims.Width
		fitWidth = &o.Height
		fitHeight = &o.Width
	}

	*fitWidth, *fitHeight = calculateDestinationFitDimension(originWidth, originHeight, *fitWidth, *fitHeight)

	opts := BimgOptions(o)
	opts.Embed = true

	return Process(buf, opts)
}

func calculateDestinationFitDimension(imageWidth, imageHeight, fitWidth, fitHeight int) (int, int) {
	if imageWidth*fitHeight > fitWidth*imageHeight {
		fitHeight = int(math.Round(float64(fitWidth) * float64(imageHeight) / float64(imageWidth)))
	} else {
		fitWidth = int(math.Round(float64(fitHeight) * float64(imageWidth) / float64(imageHeight)))
	}

	return fitWidth, fitHeight
}

// Enlarge resizes the image, allowing upscaling beyond the original dimensions.
func Enlarge(buf []byte, o ImageOptions) (Image, error) {
	if o.Width == 0 || o.Height == 0 {
		return Image{}, NewInvalidParamError("Missing required params: height, width")
	}

	opts := BimgOptions(o)
	opts.Enlarge = true

	if o.NoCrop != nil {
		opts.Crop = !*o.NoCrop
	}

	return Process(buf, opts)
}

// Extract crops a rectangular area from the image.
func Extract(buf []byte, o ImageOptions) (Image, error) {
	if o.AreaWidth == 0 || o.AreaHeight == 0 {
		return Image{}, NewInvalidParamError("Missing required params: areawidth or areaheight")
	}

	opts := BimgOptions(o)
	opts.Top = o.Top
	opts.Left = o.Left
	opts.AreaWidth = o.AreaWidth
	opts.AreaHeight = o.AreaHeight

	return Process(buf, opts)
}

// Crop crops the image to the given width and height.
func Crop(buf []byte, o ImageOptions) (Image, error) {
	if o.Width == 0 && o.Height == 0 {
		return Image{}, NewInvalidParamError("Missing required param: height or width")
	}

	opts := BimgOptions(o)
	opts.Crop = true
	return Process(buf, opts)
}

// SmartCrop crops the image using content-aware smart cropping.
func SmartCrop(buf []byte, o ImageOptions) (Image, error) {
	if o.Width == 0 && o.Height == 0 {
		return Image{}, NewInvalidParamError("Missing required param: height or width")
	}

	opts := BimgOptions(o)
	opts.Crop = true
	opts.Gravity = bimg.GravitySmart
	return Process(buf, opts)
}

// Rotate rotates the image by the given angle.
func Rotate(buf []byte, o ImageOptions) (Image, error) {
	if o.Rotate == 0 {
		return Image{}, NewInvalidParamError("Missing required param: rotate")
	}

	opts := BimgOptions(o)
	return Process(buf, opts)
}

// AutoRotate automatically rotates the image based on its EXIF orientation.
func AutoRotate(buf []byte, o ImageOptions) (out Image, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch value := r.(type) {
			case error:
				err = WrapProcessingError("libvips processing error", value)
			case string:
				err = NewProcessingError(value)
			default:
				err = NewProcessingError("libvips internal error")
			}
			out = Image{}
		}
	}()

	ibuf, err := bimg.NewImage(buf).AutoRotate()
	if err != nil {
		return Image{}, WrapProcessingError("auto-rotate failed", err)
	}

	mime := GetImageMimeType(bimg.DetermineImageType(ibuf))
	return Image{Body: ibuf, Mime: mime}, nil
}

// Flip flips the image vertically.
func Flip(buf []byte, o ImageOptions) (Image, error) {
	opts := BimgOptions(o)
	opts.Flip = true
	return Process(buf, opts)
}

// Flop flips the image horizontally.
func Flop(buf []byte, o ImageOptions) (Image, error) {
	opts := BimgOptions(o)
	opts.Flop = true
	return Process(buf, opts)
}

// Thumbnail generates a thumbnail of the image.
func Thumbnail(buf []byte, o ImageOptions) (Image, error) {
	if o.Width == 0 && o.Height == 0 {
		return Image{}, NewInvalidParamError("Missing required params: width or height")
	}

	return Process(buf, BimgOptions(o))
}

// Zoom zooms the image by the given factor.
func Zoom(buf []byte, o ImageOptions) (Image, error) {
	if o.Factor == 0 {
		return Image{}, NewInvalidParamError("Missing required param: factor")
	}

	opts := BimgOptions(o)

	if o.Top > 0 || o.Left > 0 {
		if o.AreaWidth == 0 && o.AreaHeight == 0 {
			return Image{}, NewInvalidParamError("Missing required params: areawidth, areaheight")
		}

		opts.Top = o.Top
		opts.Left = o.Left
		opts.AreaWidth = o.AreaWidth
		opts.AreaHeight = o.AreaHeight

		if o.NoCrop != nil {
			opts.Crop = !*o.NoCrop
		}
	}

	opts.Zoom = o.Factor
	return Process(buf, opts)
}

// Convert converts the image to the specified output format.
func Convert(buf []byte, o ImageOptions) (Image, error) {
	if o.Type == "" {
		return Image{}, NewInvalidParamError("Missing required param: type")
	}
	if ImageType(o.Type) == bimg.UNKNOWN {
		return Image{}, NewInvalidParamError("Invalid image type: " + o.Type)
	}
	opts := BimgOptions(o)

	return Process(buf, opts)
}

// Watermark adds a text watermark to the image.
func Watermark(buf []byte, o ImageOptions) (Image, error) {
	if o.Text == "" {
		return Image{}, NewInvalidParamError("Missing required param: text")
	}

	opts := BimgOptions(o)
	opts.Watermark.DPI = o.DPI
	opts.Watermark.Text = o.Text
	opts.Watermark.Font = o.Font
	opts.Watermark.Margin = o.Margin
	opts.Watermark.Width = o.TextWidth
	opts.Watermark.Opacity = o.Opacity
	opts.Watermark.NoReplicate = derefBool(o.NoReplicate, false)

	if len(o.Color) > 2 {
		opts.Watermark.Background = bimg.Color{R: o.Color[0], G: o.Color[1], B: o.Color[2]}
	}

	return Process(buf, opts)
}

// WatermarkImage composites a watermark image onto the source image.
func WatermarkImage(buf []byte, o ImageOptions) (Image, error) {
	if len(o.ImageBytes) == 0 {
		return Image{}, NewInvalidParamError("Missing required param: image")
	}

	opts := BimgOptions(o)
	opts.WatermarkImage.Left = o.Left
	opts.WatermarkImage.Top = o.Top
	opts.WatermarkImage.Buf = o.ImageBytes
	opts.WatermarkImage.Opacity = o.Opacity

	return Process(buf, opts)
}

// GaussianBlur applies a Gaussian blur to the image.
func GaussianBlur(buf []byte, o ImageOptions) (Image, error) {
	if o.Sigma == 0 && o.MinAmpl == 0 {
		return Image{}, NewInvalidParamError("Missing required param: sigma or minampl")
	}
	opts := BimgOptions(o)
	return Process(buf, opts)
}

// Pipeline applies a sequence of image operations.
func Pipeline(buf []byte, o ImageOptions) (Image, error) {
	if len(o.Operations) == 0 {
		return Image{}, NewInvalidParamError("Missing or invalid pipeline operations JSON")
	}
	if len(o.Operations) > 10 {
		return Image{}, NewInvalidParamError("Maximum allowed pipeline operations exceeded")
	}

	for i, operation := range o.Operations {
		var exists bool
		if operation.Operation, exists = OperationsMap[operation.Name]; !exists {
			return Image{}, NewInvalidParamError(fmt.Sprintf("Unsupported operation name: %s", operation.Name))
		}

		var err error
		operation.ImageOptions, err = BuildParamsFromOperation(operation)
		if err != nil {
			return Image{}, err
		}

		o.Operations[i] = operation
	}

	var image Image
	var err error

	image = Image{Body: buf}
	for _, operation := range o.Operations {
		var curImage Image
		curImage, err = operation.Operation(image.Body, operation.ImageOptions)
		if err != nil && !operation.IgnoreFailure {
			return Image{}, err
		}
		if operation.IgnoreFailure {
			err = nil
		}
		if err == nil {
			image = curImage
		}
	}

	return image, err
}

// Process applies bimg resize options to the image buffer.
func Process(buf []byte, opts bimg.Options) (out Image, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch value := r.(type) {
			case error:
				err = WrapProcessingError("libvips processing error", value)
			case string:
				err = NewProcessingError(value)
			default:
				err = NewProcessingError("libvips internal error")
			}
			out = Image{}
		}
	}()

	ibuf, err := bimg.Resize(buf, opts)

	if err != nil && strings.Contains(err.Error(), "encode") && (opts.Type == bimg.WEBP || opts.Type == bimg.HEIF) {
		opts.Type = bimg.JPEG
		ibuf, err = bimg.Resize(buf, opts)
	}

	if err != nil {
		return Image{}, WrapProcessingError("error processing image", err)
	}

	mime := GetImageMimeType(bimg.DetermineImageType(ibuf))
	return Image{Body: ibuf, Mime: mime}, nil
}
