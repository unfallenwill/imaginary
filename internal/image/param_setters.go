package image

// paramSetter applies a parsed query parameter value to the appropriate field
// on ImageOptions. Each setter is responsible for parsing and validation.
type paramSetter func(opts *ImageOptions, value string) error

// paramIntField returns a paramSetter that parses value as int and assigns it
// via the provided closure.
func paramIntField(set func(*ImageOptions, int)) paramSetter {
	return func(opts *ImageOptions, value string) error {
		v, err := parseInt(value)
		set(opts, v)
		return err
	}
}

// paramFloat32Field returns a paramSetter for float32 fields.
func paramFloat32Field(set func(*ImageOptions, float32)) paramSetter {
	return func(opts *ImageOptions, value string) error {
		v, err := parseFloat(value)
		set(opts, float32(v))
		return err
	}
}

// paramFloat64Field returns a paramSetter for float64 fields.
func paramFloat64Field(set func(*ImageOptions, float64)) paramSetter {
	return func(opts *ImageOptions, value string) error {
		v, err := parseFloat(value)
		set(opts, v)
		return err
	}
}

// paramBoolPtrField returns a paramSetter for *bool fields.
func paramBoolPtrField(set func(*ImageOptions, *bool)) paramSetter {
	return func(opts *ImageOptions, value string) error {
		v, err := parseBool(value)
		set(opts, boolPtr(v))
		return err
	}
}

// paramStringField returns a paramSetter for direct string assignment.
func paramStringField(set func(*ImageOptions, string)) paramSetter {
	return func(opts *ImageOptions, value string) error {
		set(opts, value)
		return nil
	}
}

// paramSetters maps query parameter names to their setter functions.
// Unknown keys are absent from this map and silently ignored.
var paramSetters = map[string]paramSetter{
	// Integer fields
	"width":       paramIntField(func(o *ImageOptions, v int) { o.Width = v }),
	"height":      paramIntField(func(o *ImageOptions, v int) { o.Height = v }),
	"quality":     paramIntField(func(o *ImageOptions, v int) { o.Quality = v }),
	"top":         paramIntField(func(o *ImageOptions, v int) { o.Top = v }),
	"left":        paramIntField(func(o *ImageOptions, v int) { o.Left = v }),
	"areawidth":   paramIntField(func(o *ImageOptions, v int) { o.AreaWidth = v }),
	"areaheight":  paramIntField(func(o *ImageOptions, v int) { o.AreaHeight = v }),
	"compression": paramIntField(func(o *ImageOptions, v int) { o.Compression = v }),
	"rotate":      paramIntField(func(o *ImageOptions, v int) { o.Rotate = v }),
	"margin":      paramIntField(func(o *ImageOptions, v int) { o.Margin = v }),
	"factor":      paramIntField(func(o *ImageOptions, v int) { o.Factor = v }),
	"dpi":         paramIntField(func(o *ImageOptions, v int) { o.DPI = v }),
	"textwidth":   paramIntField(func(o *ImageOptions, v int) { o.TextWidth = v }),
	"speed":       paramIntField(func(o *ImageOptions, v int) { o.Speed = v }),

	// Float fields
	"opacity": paramFloat32Field(func(o *ImageOptions, v float32) { o.Opacity = v }),
	"sigma":   paramFloat64Field(func(o *ImageOptions, v float64) { o.Sigma = v }),
	"minampl": paramFloat64Field(func(o *ImageOptions, v float64) { o.MinAmpl = v }),

	// Boolean pointer fields
	"flip":        paramBoolPtrField(func(o *ImageOptions, v *bool) { o.Flip = v }),
	"flop":        paramBoolPtrField(func(o *ImageOptions, v *bool) { o.Flop = v }),
	"nocrop":      paramBoolPtrField(func(o *ImageOptions, v *bool) { o.NoCrop = v }),
	"noprofile":   paramBoolPtrField(func(o *ImageOptions, v *bool) { o.NoProfile = v }),
	"norotation":  paramBoolPtrField(func(o *ImageOptions, v *bool) { o.NoRotation = v }),
	"noreplicate": paramBoolPtrField(func(o *ImageOptions, v *bool) { o.NoReplicate = v }),
	"force":       paramBoolPtrField(func(o *ImageOptions, v *bool) { o.Force = v }),
	"embed":       paramBoolPtrField(func(o *ImageOptions, v *bool) { o.Embed = v }),
	"stripmeta":   paramBoolPtrField(func(o *ImageOptions, v *bool) { o.StripMetadata = v }),
	"interlace":   paramBoolPtrField(func(o *ImageOptions, v *bool) { o.Interlace = v }),
	"palette":     paramBoolPtrField(func(o *ImageOptions, v *bool) { o.Palette = v }),

	// String fields
	"text":        paramStringField(func(o *ImageOptions, v string) { o.Text = v }),
	"image":       paramStringField(func(o *ImageOptions, v string) { o.Image = v }),
	"font":        paramStringField(func(o *ImageOptions, v string) { o.Font = v }),
	"type":        paramStringField(func(o *ImageOptions, v string) { o.Type = v }),
	"aspectratio": paramStringField(func(o *ImageOptions, v string) { o.AspectRatio = v }),

	// Parsed string fields (string -> bimg type)
	"color":      paramStringField(func(o *ImageOptions, v string) { o.Color = parseColor(v) }),
	"background": paramStringField(func(o *ImageOptions, v string) { o.Background = parseColor(v) }),
	"colorspace": paramStringField(func(o *ImageOptions, v string) { o.Colorspace = parseColorspace(v) }),
	"gravity":    paramStringField(func(o *ImageOptions, v string) { o.Gravity = parseGravity(v) }),
	"extend":     paramStringField(func(o *ImageOptions, v string) { o.Extend = parseExtendMode(v) }),

	// Composite fields
	"operations": func(opts *ImageOptions, value string) error {
		ops, err := parseJSONOperations(value)
		if err != nil {
			return err
		}
		opts.Operations = ops
		return nil
	},
}
