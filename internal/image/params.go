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
	setter, ok := paramSetters[key]
	if !ok {
		return nil
	}
	return setter(opts, value)
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
