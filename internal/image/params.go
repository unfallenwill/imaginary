package image

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/h2non/bimg"
)

// BuildParamsFromQuery builds ImageOptions from HTTP query parameters.
// Unknown parameter names are silently ignored, preserving backward compatibility.
func BuildParamsFromQuery(query map[string][]string) (ImageOptions, error) {
	var options ImageOptions
	options.Extend = bimg.ExtendCopy

	for key, vals := range query {
		value := ""
		if len(vals) > 0 {
			value = vals[0]
		}
		if err := applyQueryParam(&options, key, value); err != nil {
			return ImageOptions{}, fmt.Errorf(`error while processing parameter "%s" with value %q, error: %w`, key, value, err)
		}
	}

	return options, nil
}

// BuildParamsFromOperation builds ImageOptions from a pipeline operation's params.
func BuildParamsFromOperation(op PipelineOperation) (ImageOptions, error) {
	return op.Params.ToImageOptions(), nil
}

// applyQueryParam sets the appropriate field on ImageOptions for a known query parameter.
// Returns ErrUnsupportedValue for malformed values. Unknown keys return nil (ignored).
func applyQueryParam(opts *ImageOptions, key, value string) error {
	switch key {
	// Integer fields
	case "width":
		v, err := parseInt(value)
		opts.Width = v
		return err
	case "height":
		v, err := parseInt(value)
		opts.Height = v
		return err
	case "quality":
		v, err := parseInt(value)
		opts.Quality = v
		return err
	case "top":
		v, err := parseInt(value)
		opts.Top = v
		return err
	case "left":
		v, err := parseInt(value)
		opts.Left = v
		return err
	case "areawidth":
		v, err := parseInt(value)
		opts.AreaWidth = v
		return err
	case "areaheight":
		v, err := parseInt(value)
		opts.AreaHeight = v
		return err
	case "compression":
		v, err := parseInt(value)
		opts.Compression = v
		return err
	case "rotate":
		v, err := parseInt(value)
		opts.Rotate = v
		return err
	case "margin":
		v, err := parseInt(value)
		opts.Margin = v
		return err
	case "factor":
		v, err := parseInt(value)
		opts.Factor = v
		return err
	case "dpi":
		v, err := parseInt(value)
		opts.DPI = v
		return err
	case "textwidth":
		v, err := parseInt(value)
		opts.TextWidth = v
		return err
	case "speed":
		v, err := parseInt(value)
		opts.Speed = v
		return err

	// Float fields
	case "opacity":
		v, err := parseFloat(value)
		opts.Opacity = float32(v)
		return err
	case "sigma":
		v, err := parseFloat(value)
		opts.Sigma = v
		return err
	case "minampl":
		v, err := parseFloat(value)
		opts.MinAmpl = v
		return err

	// Boolean pointer fields
	case "flip":
		v, err := parseBool(value)
		opts.Flip = boolPtr(v)
		return err
	case "flop":
		v, err := parseBool(value)
		opts.Flop = boolPtr(v)
		return err
	case "nocrop":
		v, err := parseBool(value)
		opts.NoCrop = boolPtr(v)
		return err
	case "noprofile":
		v, err := parseBool(value)
		opts.NoProfile = boolPtr(v)
		return err
	case "norotation":
		v, err := parseBool(value)
		opts.NoRotation = boolPtr(v)
		return err
	case "noreplicate":
		v, err := parseBool(value)
		opts.NoReplicate = boolPtr(v)
		return err
	case "force":
		v, err := parseBool(value)
		opts.Force = boolPtr(v)
		return err
	case "embed":
		v, err := parseBool(value)
		opts.Embed = boolPtr(v)
		return err
	case "stripmeta":
		v, err := parseBool(value)
		opts.StripMetadata = boolPtr(v)
		return err
	case "interlace":
		v, err := parseBool(value)
		opts.Interlace = boolPtr(v)
		return err
	case "palette":
		v, err := parseBool(value)
		opts.Palette = boolPtr(v)
		return err

	// String fields
	case "text":
		opts.Text = value
		return nil
	case "image":
		opts.Image = value
		return nil
	case "font":
		opts.Font = value
		return nil
	case "type":
		opts.Type = value
		return nil
	case "aspectratio":
		opts.AspectRatio = value
		return nil

	// Parsed string fields (string -> bimg type)
	case "color":
		opts.Color = parseColor(value)
		return nil
	case "background":
		opts.Background = parseColor(value)
		return nil
	case "colorspace":
		opts.Colorspace = parseColorspace(value)
		return nil
	case "gravity":
		opts.Gravity = parseGravity(value)
		return nil
	case "extend":
		opts.Extend = parseExtendMode(value)
		return nil

	// Composite fields
	case "operations":
		ops, err := parseJSONOperations(value)
		if err != nil {
			return err
		}
		opts.Operations = ops
		return nil

	default:
		return nil
	}
}

func parseBool(val string) (bool, error) {
	if val == "" {
		return false, nil
	}

	return strconv.ParseBool(val)
}

func parseInt(param string) (int, error) {
	if param == "" {
		return 0, nil
	}

	f, err := parseFloat(param)
	return int(math.Floor(f + 0.5)), err
}

func parseFloat(param string) (float64, error) {
	if param == "" {
		return 0.0, nil
	}

	val, err := strconv.ParseFloat(param, 64)
	return math.Abs(val), err
}

func parseColorspace(val string) bimg.Interpretation {
	if val == "bw" {
		return bimg.InterpretationBW
	}
	return bimg.InterpretationSRGB
}

func parseColor(val string) []uint8 {
	const max float64 = 255
	var buf []uint8
	if val != "" {
		for _, num := range strings.Split(val, ",") {
			n, _ := strconv.ParseUint(strings.Trim(num, " "), 10, 8)
			buf = append(buf, uint8(math.Min(float64(n), max)))
		}
	}
	return buf
}

func parseJSONOperations(data string) (PipelineOperations, error) {
	var operations PipelineOperations

	// Fewer than 2 characters cannot be valid JSON. We assume empty operation.
	if len(data) < 2 {
		return operations, nil
	}

	d := json.NewDecoder(strings.NewReader(data))

	err := d.Decode(&operations)
	return operations, err
}

func parseExtendMode(val string) bimg.Extend {
	var m = map[string]bimg.Extend{
		"white":      bimg.ExtendWhite,
		"black":      bimg.ExtendBlack,
		"copy":       bimg.ExtendCopy,
		"background": bimg.ExtendBackground,
		"lastpixel":  bimg.ExtendLast,
		"mirror":     bimg.ExtendMirror,
	}

	val = strings.TrimSpace(strings.ToLower(val))
	if e, ok := m[val]; ok {
		return e
	}
	return bimg.ExtendMirror
}

func parseGravity(val string) bimg.Gravity {
	var m = map[string]bimg.Gravity{
		"south": bimg.GravitySouth,
		"north": bimg.GravityNorth,
		"east":  bimg.GravityEast,
		"west":  bimg.GravityWest,
		"smart": bimg.GravitySmart,
	}

	val = strings.TrimSpace(strings.ToLower(val))
	if g, ok := m[val]; ok {
		return g
	}

	return bimg.GravityCentre
}
