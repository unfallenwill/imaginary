package image

import (
	"math"
	"strconv"
	"strings"

	"github.com/h2non/bimg"
)

// BuildParamsFromOperation builds ImageOptions from a pipeline operation's params.
func BuildParamsFromOperation(op PipelineOperation) (ImageOptions, error) {
	return op.Params.ToImageOptions(), nil
}

// ParseColor converts a comma-separated RGB string to a uint8 slice.
func ParseColor(val string) []uint8 {
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

// ParseColorspace converts a string to a bimg color interpretation.
func ParseColorspace(val string) bimg.Interpretation {
	if val == "bw" {
		return bimg.InterpretationBW
	}
	return bimg.InterpretationSRGB
}

// ParseExtendMode converts a string to a bimg extend mode.
func ParseExtendMode(val string) bimg.Extend {
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

// ParseGravity converts a string to a bimg gravity value.
func ParseGravity(val string) bimg.Gravity {
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
