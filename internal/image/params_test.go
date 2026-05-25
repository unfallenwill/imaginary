package image

import (
	"math"
	"testing"

	"github.com/h2non/bimg"
)

const epsilon = 0.0001

func TestReadParams(t *testing.T) {
	q := map[string][]string{
		"width":       {"100"},
		"height":      {"80"},
		"noreplicate": {"1"},
		"opacity":     {"0.2"},
		"text":        {"hello"},
		"background":  {"255,10,20"},
		"interlace":   {"true"},
	}

	params, err := BuildParamsFromQuery(q)
	if err != nil {
		t.Errorf("Failed reading params, %s", err)
	}

	assert := params.Width == 100 &&
		params.Height == 80 &&
		derefBool(params.NoReplicate, false) == true &&
		params.Opacity == 0.2 &&
		params.Text == "hello" &&
		params.Background[0] == 255 &&
		params.Background[1] == 10 &&
		params.Background[2] == 20 &&
		derefBool(params.Interlace, false) == true

	if assert == false {
		t.Error("Invalid params")
	}
}

func TestParseParam(t *testing.T) {
	intCases := []struct {
		value    string
		expected int
	}{
		{"1", 1},
		{"0100", 100},
		{"-100", 100},
		{"99.02", 99},
		{"99.9", 100},
	}

	for _, test := range intCases {
		val, _ := parseInt(test.value)
		if val != test.expected {
			t.Errorf("Invalid param: %s != %d", test.value, test.expected)
		}
	}

	floatCases := []struct {
		value    string
		expected float64
	}{
		{"1.1", 1.1},
		{"01.1", 1.1},
		{"-1.10", 1.10},
		{"99.999999", 99.999999},
	}

	for _, test := range floatCases {
		val, _ := parseFloat(test.value)
		if val != test.expected {
			t.Errorf("Invalid param: %#v != %#v", val, test.expected)
		}
	}

	boolCases := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"1", true},
		{"1.1", false},
		{"-1", false},
		{"0", false},
		{"0.0", false},
		{"no", false},
		{"yes", false},
	}

	for _, test := range boolCases {
		val, _ := parseBool(test.value)
		if val != test.expected {
			t.Errorf("Invalid param: %#v != %#v", val, test.expected)
		}
	}
}

func TestParseColor(t *testing.T) {
	cases := []struct {
		value    string
		expected []uint8
	}{
		{"200,100,20", []uint8{200, 100, 20}},
		{"0,280,200", []uint8{0, 255, 200}},
		{" -1, 256 , 50", []uint8{0, 255, 50}},
		{" a, 20 , &hel0", []uint8{0, 20, 0}},
		{"", []uint8{}},
	}

	for _, color := range cases {
		c := parseColor(color.value)
		l := len(color.expected)

		if len(c) != l {
			t.Errorf("Invalid color length: %#v", c)
		}
		if l == 0 {
			continue
		}

		assert := c[0] == color.expected[0] &&
			c[1] == color.expected[1] &&
			c[2] == color.expected[2]

		if assert == false {
			t.Errorf("Invalid color schema: %#v <> %#v", color.expected, c)
		}
	}
}

func TestParseExtend(t *testing.T) {
	cases := []struct {
		value    string
		expected bimg.Extend
	}{
		{"white", bimg.ExtendWhite},
		{"black", bimg.ExtendBlack},
		{"copy", bimg.ExtendCopy},
		{"mirror", bimg.ExtendMirror},
		{"lastpixel", bimg.ExtendLast},
		{"background", bimg.ExtendBackground},
		{" BACKGROUND  ", bimg.ExtendBackground},
		{"invalid", bimg.ExtendMirror},
		{"", bimg.ExtendMirror},
	}

	for _, extend := range cases {
		c := parseExtendMode(extend.value)
		if c != extend.expected {
			t.Errorf("Invalid extend value : %d != %d", c, extend.expected)
		}
	}
}

func TestGravity(t *testing.T) {
	cases := []struct {
		gravityValue   string
		smartCropValue bool
	}{
		{gravityValue: "foo", smartCropValue: false},
		{gravityValue: "smart", smartCropValue: true},
	}

	for _, td := range cases {
		io, _ := BuildParamsFromQuery(map[string][]string{"gravity": {td.gravityValue}})
		if (io.Gravity == bimg.GravitySmart) != td.smartCropValue {
			t.Errorf("Expected %t to be %t, test data: %+v", io.Gravity == bimg.GravitySmart, td.smartCropValue, td)
		}
	}
}

func TestBuildParamsFromOperation(t *testing.T) {
	pp := PipelineParams{
		Width:      intPtr(200),
		Opacity:    floatPtr(2.2),
		Force:      boolPtr(true),
		StripMeta:  boolPtr(false),
		Type:       "jpeg",
		Background: "255,12,3",
	}

	op := PipelineOperation{Params: pp}
	options, err := BuildParamsFromOperation(op)
	if err != nil {
		t.Errorf("Expected this to work! %s", err)
	}

	if options.Width != 200 {
		t.Errorf("Expected the Width to be coerced with the correct value of %d", 200)
	}

	if math.Abs(float64(options.Opacity)-2.2) > epsilon {
		t.Errorf("Expected the Opacity to be coerced with the correct value of %f", 2.2)
	}

	if derefBool(options.Force, false) != true || derefBool(options.StripMetadata, false) != false {
		t.Errorf("Expected boolean parameters to result in their respective value's\n%+v", options)
	}

	if options.Background[0] != 255 {
		t.Errorf("Expected color parameter to be coerced with the correct value")
	}
}

func TestPipelineParamsBoolNilVsValue(t *testing.T) {
	pp := PipelineParams{
		Flip:  boolPtr(true),
		Flop:  boolPtr(false),
		Force: nil,
	}

	op := PipelineOperation{Params: pp}
	options, err := BuildParamsFromOperation(op)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if options.Flip == nil || *options.Flip != true {
		t.Errorf("Expected Flip to be *true, got %v", options.Flip)
	}
	if options.Flop == nil || *options.Flop != false {
		t.Errorf("Expected Flop to be *false, got %v", options.Flop)
	}
	if options.Force != nil {
		t.Errorf("Expected Force to be nil (not set), got %v", options.Force)
	}
}

func TestPipelineParamsPaletteFalse(t *testing.T) {
	pp := PipelineParams{
		Palette: boolPtr(false),
	}

	op := PipelineOperation{Params: pp}
	options, err := BuildParamsFromOperation(op)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if options.Palette == nil || *options.Palette != false {
		t.Errorf("Expected Palette to be *false (bug fix), got %v", options.Palette)
	}
}

func TestParseFunctions(t *testing.T) {
	t.Run("parseBool", func(t *testing.T) {
		if r, err := parseBool("true"); r != true {
			t.Errorf("Expected string true to result a native type true %s", err)
		}

		if r, err := parseBool("false"); r != false {
			t.Errorf("Expected string false to result a native type false %s", err)
		}

		// A special case that we support
		if _, err := parseBool(""); err != nil {
			t.Errorf("Expected blank values to default to false, it didn't! %s", err)
		}

		if r, err := parseBool("foo"); err == nil {
			t.Errorf("Expected malformed values to result in an error, it didn't! %+v", r)
		}
	})
}

func TestApplyQueryParam(t *testing.T) {
	t.Run("integer fields", func(t *testing.T) {
		cases := []struct {
			key   string
			value string
			field string
			want  int
		}{
			{"width", "100", "Width", 100},
			{"height", "200", "Height", 200},
			{"quality", "95", "Quality", 95},
			{"rotate", "90", "Rotate", 90},
			{"speed", "5", "Speed", 5},
		}

		for _, tc := range cases {
			var opts ImageOptions
			err := applyQueryParam(&opts, tc.key, tc.value)
			if err != nil {
				t.Errorf("applyQueryParam(%q, %q) returned error: %s", tc.key, tc.value, err)
			}
		}
	})

	t.Run("bool pointer fields", func(t *testing.T) {
		var opts ImageOptions
		err := applyQueryParam(&opts, "flip", "true")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if opts.Flip == nil || *opts.Flip != true {
			t.Errorf("Expected Flip = *true, got %v", opts.Flip)
		}

		err = applyQueryParam(&opts, "nocrop", "false")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if opts.NoCrop == nil || *opts.NoCrop != false {
			t.Errorf("Expected NoCrop = *false, got %v", opts.NoCrop)
		}
	})

	t.Run("palette=false is respected", func(t *testing.T) {
		var opts ImageOptions
		err := applyQueryParam(&opts, "palette", "false")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if opts.Palette == nil || *opts.Palette != false {
			t.Errorf("Expected Palette = *false (bug fix), got %v", opts.Palette)
		}
	})

	t.Run("unknown key is ignored", func(t *testing.T) {
		var opts ImageOptions
		err := applyQueryParam(&opts, "unknownparam", "value")
		if err != nil {
			t.Errorf("Expected unknown key to be ignored, got error: %s", err)
		}
	})

	t.Run("parsed string fields", func(t *testing.T) {
		var opts ImageOptions
		err := applyQueryParam(&opts, "gravity", "west")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if opts.Gravity != bimg.GravityWest {
			t.Errorf("Expected GravityWest, got %d", opts.Gravity)
		}

		err = applyQueryParam(&opts, "extend", "copy")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if opts.Extend != bimg.ExtendCopy {
			t.Errorf("Expected ExtendCopy, got %d", opts.Extend)
		}
	})
}

func TestDerefBool(t *testing.T) {
	if derefBool(nil, false) != false {
		t.Error("Expected nil to return default false")
	}
	if derefBool(nil, true) != true {
		t.Error("Expected nil to return default true")
	}
	tr := true
	fl := false
	if derefBool(&tr, false) != true {
		t.Error("Expected &true to return true")
	}
	if derefBool(&fl, true) != false {
		t.Error("Expected &false to return false")
	}
}

// intPtr returns a pointer to the given int value.
func intPtr(v int) *int {
	return &v
}

// floatPtr returns a pointer to the given float64 value.
func floatPtr(v float64) *float64 {
	return &v
}
