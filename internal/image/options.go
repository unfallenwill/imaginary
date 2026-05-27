package image

import (
	"strconv"
	"strings"

	"github.com/h2non/bimg"
)

// Dimensions groups size and position related fields.
type Dimensions struct {
	Width      int
	Height     int
	AreaWidth  int
	AreaHeight int
	Top        int
	Left       int
	Margin     int
}

// Quality groups quality and compression related fields.
type Quality struct {
	Quality     int
	Compression int
	Speed       int
}

// Effects groups visual effect related fields.
type Effects struct {
	Sigma   float64
	MinAmpl float64
	Opacity float32
}

// WatermarkOpts groups watermark text and image related fields.
type WatermarkOpts struct {
	Text       string
	Font       string
	DPI        int
	TextWidth  int
	Image      string
	ImageBytes []byte
	Color      []uint8
}

// Flags groups boolean toggle fields.
type Flags struct {
	Flip          *bool
	Flop          *bool
	Force         *bool
	Embed         *bool
	NoCrop        *bool
	NoReplicate   *bool
	NoRotation    *bool
	NoProfile     *bool
	StripMetadata *bool
	Interlace     *bool
	Palette       *bool
}

// Transform groups transformation related fields.
type Transform struct {
	Rotate      int
	Factor      int
	AspectRatio string
	Extend      bimg.Extend
	Gravity     bimg.Gravity
	Colorspace  bimg.Interpretation
	Background  []uint8
}

// PipelineOpts groups pipeline operations.
type PipelineOpts struct {
	Operations PipelineOperations
}

// ImageOptions represent all the supported image transformation params.
// Boolean fields use *bool so that "explicitly set to false" can be distinguished from "not set".
type ImageOptions struct {
	Dimensions
	Quality
	Effects
	WatermarkOpts
	Flags
	Transform
	PipelineOpts
	Type string
}

// boolPtr returns a pointer to the given bool value.
func boolPtr(b bool) *bool {
	return &b
}

// derefBool returns the value pointed to by p, or defaultVal if p is nil.
func derefBool(p *bool, defaultVal bool) bool {
	if p == nil {
		return defaultVal
	}
	return *p
}

// PipelineParams holds typed parameters for a pipeline operation,
// deserialized directly from JSON. Pointer fields distinguish "not provided" from
// "provided as zero/empty".
type PipelineParams struct {
	Width       *int `json:"width"`
	Height      *int `json:"height"`
	Top         *int `json:"top"`
	Left        *int `json:"left"`
	AreaWidth   *int `json:"areawidth"`
	AreaHeight  *int `json:"areaheight"`
	Quality     *int `json:"quality"`
	Compression *int `json:"compression"`
	Rotate      *int `json:"rotate"`
	Margin      *int `json:"margin"`
	Factor      *int `json:"factor"`
	DPI         *int `json:"dpi"`
	TextWidth   *int `json:"textwidth"`
	Speed       *int `json:"speed"`

	Opacity *float64 `json:"opacity"`
	Sigma   *float64 `json:"sigma"`
	MinAmpl *float64 `json:"minampl"`

	Flip        *bool `json:"flip"`
	Flop        *bool `json:"flop"`
	NoCrop      *bool `json:"nocrop"`
	NoProfile   *bool `json:"noprofile"`
	NoRotation  *bool `json:"norotation"`
	NoReplicate *bool `json:"noreplicate"`
	Force       *bool `json:"force"`
	Embed       *bool `json:"embed"`
	StripMeta   *bool `json:"stripmeta"`
	Interlace   *bool `json:"interlace"`
	Palette     *bool `json:"palette"`

	Text        string `json:"text"`
	Image       string `json:"image"`
	ImageBytes  []byte `json:"-"`
	Font        string `json:"font"`
	Type        string `json:"type"`
	AspectRatio string `json:"aspectratio"`
	Color       string `json:"color"`
	Background  string `json:"background"`
	Colorspace  string `json:"colorspace"`
	Gravity     string `json:"gravity"`
	Extend      string `json:"extend"`
}

// derefInt returns the value pointed to by p, or 0 if p is nil.
func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// derefFloat64 returns the value pointed to by p, or 0.0 if p is nil.
func derefFloat64(p *float64) float64 {
	if p == nil {
		return 0.0
	}
	return *p
}

// ToImageOptions converts PipelineParams to ImageOptions, applying defaults.
func (p PipelineParams) ToImageOptions() ImageOptions {
	opts := ImageOptions{
		Dimensions: Dimensions{
			Width:      derefInt(p.Width),
			Height:     derefInt(p.Height),
			Top:        derefInt(p.Top),
			Left:       derefInt(p.Left),
			AreaWidth:  derefInt(p.AreaWidth),
			AreaHeight: derefInt(p.AreaHeight),
			Margin:     derefInt(p.Margin),
		},
		Quality: Quality{
			Quality:     derefInt(p.Quality),
			Compression: derefInt(p.Compression),
			Speed:       derefInt(p.Speed),
		},
		Transform: Transform{
			Extend:      bimg.ExtendCopy,
			Rotate:      derefInt(p.Rotate),
			Factor:      derefInt(p.Factor),
			AspectRatio: p.AspectRatio,
		},
		Effects: Effects{
			Opacity: float32(derefFloat64(p.Opacity)),
			Sigma:   derefFloat64(p.Sigma),
			MinAmpl: derefFloat64(p.MinAmpl),
		},
		Flags: Flags{
			Flip:          p.Flip,
			Flop:          p.Flop,
			NoCrop:        p.NoCrop,
			NoProfile:     p.NoProfile,
			NoRotation:    p.NoRotation,
			NoReplicate:   p.NoReplicate,
			Force:         p.Force,
			Embed:         p.Embed,
			StripMetadata: p.StripMeta,
			Interlace:     p.Interlace,
			Palette:       p.Palette,
		},
		WatermarkOpts: WatermarkOpts{
			Text:       p.Text,
			Image:      p.Image,
			ImageBytes: p.ImageBytes,
			Font:       p.Font,
			DPI:        derefInt(p.DPI),
			TextWidth:  derefInt(p.TextWidth),
		},
		Type: p.Type,
	}

	if p.Color != "" {
		opts.Color = ParseColor(p.Color)
	}
	if p.Background != "" {
		opts.Background = ParseColor(p.Background)
	}
	if p.Colorspace != "" {
		opts.Colorspace = ParseColorspace(p.Colorspace)
	}
	if p.Gravity != "" {
		opts.Gravity = ParseGravity(p.Gravity)
	}
	if p.Extend != "" {
		opts.Extend = ParseExtendMode(p.Extend)
	}

	return opts
}

// PipelineOperation represents the structure for an operation field.
type PipelineOperation struct {
	Name          string         `json:"operation"`
	IgnoreFailure bool           `json:"ignore_failure"`
	Params        PipelineParams `json:"params"`
	ImageOptions  ImageOptions   `json:"-"`
	Operation     Operation      `json:"-"`
}

// PipelineOperations defines the expected interface for a list of operations.
type PipelineOperations []PipelineOperation

func transformByAspectRatio(width, height int, ar map[string]int) (int, int) {
	if width != 0 {
		height = width / ar["width"] * ar["height"]
	} else {
		width = height / ar["height"] * ar["width"]
	}
	return width, height
}

func parseAspectRatio(val string) map[string]int {
	val = strings.TrimSpace(strings.ToLower(val))
	slicedVal := strings.Split(val, ":")

	if len(slicedVal) < 2 {
		return nil
	}

	width, _ := strconv.Atoi(slicedVal[0])
	height, _ := strconv.Atoi(slicedVal[1])

	return map[string]int{
		"width":  width,
		"height": height,
	}
}

func shouldTransformByAspectRatio(height, width int) bool {

	// override aspect ratio parameters if width and height is given or not given at all
	if (width != 0 && height != 0) || (width == 0 && height == 0) {
		return false
	}

	return true
}

// BimgOptions creates a new bimg compatible options struct mapping the fields properly
func BimgOptions(o ImageOptions) bimg.Options {
	opts := bimg.Options{
		Width:          o.Width,
		Height:         o.Height,
		Flip:           derefBool(o.Flip, false),
		Flop:           derefBool(o.Flop, false),
		Quality:        o.Quality.Quality,
		Compression:    o.Compression,
		NoAutoRotate:   derefBool(o.NoRotation, false),
		NoProfile:      derefBool(o.NoProfile, false),
		Force:          derefBool(o.Force, false),
		Gravity:        o.Gravity,
		Embed:          derefBool(o.Embed, false),
		Extend:         o.Extend,
		Interpretation: o.Colorspace,
		StripMetadata:  derefBool(o.StripMetadata, false),
		Type:           ImageType(o.Type),
		Rotate:         bimg.Angle(o.Rotate),
		Interlace:      derefBool(o.Interlace, false),
		Palette:        derefBool(o.Palette, false),
		Speed:          o.Speed,
	}

	if len(o.Background) != 0 {
		opts.Background = bimg.Color{R: o.Background[0], G: o.Background[1], B: o.Background[2]}
	}

	if shouldTransformByAspectRatio(opts.Height, opts.Width) && o.AspectRatio != "" {
		ar := parseAspectRatio(o.AspectRatio)
		if ar != nil {
			opts.Width, opts.Height = transformByAspectRatio(opts.Width, opts.Height, ar)
		}
	}

	if o.Sigma > 0 || o.MinAmpl > 0 {
		opts.GaussianBlur = bimg.GaussianBlur{
			Sigma:   o.Sigma,
			MinAmpl: o.MinAmpl,
		}
	}

	return opts
}
