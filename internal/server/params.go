package server

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/h2non/bimg"

	img "github.com/h2non/imaginary/internal/image"
)

// BuildParamsFromQuery builds ImageOptions from HTTP query parameters.
// Unknown parameter names are silently ignored, preserving backward compatibility.
func BuildParamsFromQuery(query map[string][]string) (img.ImageOptions, error) {
	var options img.ImageOptions
	options.Extend = bimg.ExtendCopy

	for key, vals := range query {
		value := ""
		if len(vals) > 0 {
			value = vals[0]
		}
		if err := applyQueryParam(&options, key, value); err != nil {
			return img.ImageOptions{}, fmt.Errorf(`error while processing parameter "%s" with value %q, error: %w`, key, value, err)
		}
	}

	return options, nil
}

// applyQueryParam sets the appropriate field on ImageOptions for a known query parameter.
// Returns a parse error for malformed values. Unknown keys return nil (ignored).
func applyQueryParam(opts *img.ImageOptions, key, value string) error {
	setter, ok := paramSetters[key]
	if !ok {
		return nil
	}
	return setter(opts, value)
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

func parseBool(val string) (bool, error) {
	if val == "" {
		return false, nil
	}

	return strconv.ParseBool(val)
}

func parseJSONOperations(data string) (img.PipelineOperations, error) {
	var operations img.PipelineOperations

	if len(data) < 2 {
		return operations, nil
	}

	d := json.NewDecoder(strings.NewReader(data))
	err := d.Decode(&operations)
	return operations, err
}
