package server

import img "github.com/h2non/imaginary/internal/image"

// paramSetter applies a parsed query parameter value to the appropriate field
// on ImageOptions. Each setter is responsible for parsing and validation.
type paramSetter func(opts *img.ImageOptions, value string) error

// paramIntField returns a paramSetter that parses value as int and assigns it
// via the provided closure.
func paramIntField(set func(*img.ImageOptions, int)) paramSetter {
	return func(opts *img.ImageOptions, value string) error {
		v, err := parseInt(value)
		set(opts, v)
		return err
	}
}

// paramFloat32Field returns a paramSetter for float32 fields.
func paramFloat32Field(set func(*img.ImageOptions, float32)) paramSetter {
	return func(opts *img.ImageOptions, value string) error {
		v, err := parseFloat(value)
		set(opts, float32(v))
		return err
	}
}

// paramFloat64Field returns a paramSetter for float64 fields.
func paramFloat64Field(set func(*img.ImageOptions, float64)) paramSetter {
	return func(opts *img.ImageOptions, value string) error {
		v, err := parseFloat(value)
		set(opts, v)
		return err
	}
}

// paramBoolPtrField returns a paramSetter for *bool fields.
func paramBoolPtrField(set func(*img.ImageOptions, *bool)) paramSetter {
	return func(opts *img.ImageOptions, value string) error {
		v, err := parseBool(value)
		set(opts, boolPtr(v))
		return err
	}
}

// paramStringField returns a paramSetter for direct string assignment.
func paramStringField(set func(*img.ImageOptions, string)) paramSetter {
	return func(opts *img.ImageOptions, value string) error {
		set(opts, value)
		return nil
	}
}

func boolPtr(b bool) *bool {
	return &b
}

// paramSetters maps query parameter names to their setter functions.
// Unknown keys are absent from this map and silently ignored.
var paramSetters = map[string]paramSetter{
	// Integer fields
	"width":       paramIntField(func(o *img.ImageOptions, v int) { o.Width = v }),
	"height":      paramIntField(func(o *img.ImageOptions, v int) { o.Height = v }),
	"quality":     paramIntField(func(o *img.ImageOptions, v int) { o.Quality.Quality = v }),
	"top":         paramIntField(func(o *img.ImageOptions, v int) { o.Top = v }),
	"left":        paramIntField(func(o *img.ImageOptions, v int) { o.Left = v }),
	"areawidth":   paramIntField(func(o *img.ImageOptions, v int) { o.AreaWidth = v }),
	"areaheight":  paramIntField(func(o *img.ImageOptions, v int) { o.AreaHeight = v }),
	"compression": paramIntField(func(o *img.ImageOptions, v int) { o.Compression = v }),
	"rotate":      paramIntField(func(o *img.ImageOptions, v int) { o.Rotate = v }),
	"margin":      paramIntField(func(o *img.ImageOptions, v int) { o.Margin = v }),
	"factor":      paramIntField(func(o *img.ImageOptions, v int) { o.Factor = v }),
	"dpi":         paramIntField(func(o *img.ImageOptions, v int) { o.DPI = v }),
	"textwidth":   paramIntField(func(o *img.ImageOptions, v int) { o.TextWidth = v }),
	"speed":       paramIntField(func(o *img.ImageOptions, v int) { o.Speed = v }),

	// Float fields
	"opacity": paramFloat32Field(func(o *img.ImageOptions, v float32) { o.Opacity = v }),
	"sigma":   paramFloat64Field(func(o *img.ImageOptions, v float64) { o.Sigma = v }),
	"minampl": paramFloat64Field(func(o *img.ImageOptions, v float64) { o.MinAmpl = v }),

	// Boolean pointer fields
	"flip":        paramBoolPtrField(func(o *img.ImageOptions, v *bool) { o.Flip = v }),
	"flop":        paramBoolPtrField(func(o *img.ImageOptions, v *bool) { o.Flop = v }),
	"nocrop":      paramBoolPtrField(func(o *img.ImageOptions, v *bool) { o.NoCrop = v }),
	"noprofile":   paramBoolPtrField(func(o *img.ImageOptions, v *bool) { o.NoProfile = v }),
	"norotation":  paramBoolPtrField(func(o *img.ImageOptions, v *bool) { o.NoRotation = v }),
	"noreplicate": paramBoolPtrField(func(o *img.ImageOptions, v *bool) { o.NoReplicate = v }),
	"force":       paramBoolPtrField(func(o *img.ImageOptions, v *bool) { o.Force = v }),
	"embed":       paramBoolPtrField(func(o *img.ImageOptions, v *bool) { o.Embed = v }),
	"stripmeta":   paramBoolPtrField(func(o *img.ImageOptions, v *bool) { o.StripMetadata = v }),
	"interlace":   paramBoolPtrField(func(o *img.ImageOptions, v *bool) { o.Interlace = v }),
	"palette":     paramBoolPtrField(func(o *img.ImageOptions, v *bool) { o.Palette = v }),

	// String fields
	"text":        paramStringField(func(o *img.ImageOptions, v string) { o.Text = v }),
	"image":       paramStringField(func(o *img.ImageOptions, v string) { o.Image = v }),
	"font":        paramStringField(func(o *img.ImageOptions, v string) { o.Font = v }),
	"type":        paramStringField(func(o *img.ImageOptions, v string) { o.Type = v }),
	"aspectratio": paramStringField(func(o *img.ImageOptions, v string) { o.AspectRatio = v }),

	// Parsed string fields (string -> domain types, via exported image functions)
	"color":      paramStringField(func(o *img.ImageOptions, v string) { o.Color = img.ParseColor(v) }),
	"background": paramStringField(func(o *img.ImageOptions, v string) { o.Background = img.ParseColor(v) }),
	"colorspace": paramStringField(func(o *img.ImageOptions, v string) { o.Colorspace = img.ParseColorspace(v) }),
	"gravity":    paramStringField(func(o *img.ImageOptions, v string) { o.Gravity = img.ParseGravity(v) }),
	"extend":     paramStringField(func(o *img.ImageOptions, v string) { o.Extend = img.ParseExtendMode(v) }),

	// Composite fields
	"operations": func(opts *img.ImageOptions, value string) error {
		ops, err := parseJSONOperations(value)
		if err != nil {
			return err
		}
		opts.Operations = ops
		return nil
	},
}
