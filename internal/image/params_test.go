package image

import (
	"math"
	"testing"

	"github.com/h2non/bimg"
)

const epsilon = 0.0001

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
		c := ParseColor(color.value)
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
		c := ParseExtendMode(extend.value)
		if c != extend.expected {
			t.Errorf("Invalid extend value : %d != %d", c, extend.expected)
		}
	}
}

func TestGravityParse(t *testing.T) {
	cases := []struct {
		value    string
		expected bimg.Gravity
	}{
		{"south", bimg.GravitySouth},
		{"north", bimg.GravityNorth},
		{"east", bimg.GravityEast},
		{"west", bimg.GravityWest},
		{"smart", bimg.GravitySmart},
		{"foo", bimg.GravityCentre},
		{"", bimg.GravityCentre},
	}

	for _, td := range cases {
		got := ParseGravity(td.value)
		if got != td.expected {
			t.Errorf("ParseGravity(%q) = %d, want %d", td.value, got, td.expected)
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

func TestShouldTransformByAspectRatio(t *testing.T) {
	cases := []struct {
		name   string
		width  int
		height int
		want   bool
	}{
		{"BothZero", 0, 0, false},
		{"BothNonZero", 300, 200, false},
		{"WidthOnly", 300, 0, true},
		{"HeightOnly", 0, 200, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldTransformByAspectRatio(tc.height, tc.width)
			if got != tc.want {
				t.Errorf("shouldTransformByAspectRatio(height=%d, width=%d) = %v, want %v",
					tc.height, tc.width, got, tc.want)
			}
		})
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
