package server

import (
	"testing"

	"github.com/h2non/bimg"

	img "github.com/h2non/imaginary/internal/image"
)

func TestBuildParamsFromQuery(t *testing.T) {
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
		t.Fatalf("Failed reading params, %s", err)
	}

	if params.Width != 100 {
		t.Errorf("Expected Width=100, got %d", params.Width)
	}
	if params.Height != 80 {
		t.Errorf("Expected Height=80, got %d", params.Height)
	}
	if params.NoReplicate == nil || *params.NoReplicate != true {
		t.Errorf("Expected NoReplicate=true")
	}
	if params.Opacity != 0.2 {
		t.Errorf("Expected Opacity=0.2, got %f", params.Opacity)
	}
	if params.Text != "hello" {
		t.Errorf("Expected Text=hello, got %s", params.Text)
	}
	if len(params.Background) != 3 || params.Background[0] != 255 {
		t.Errorf("Expected Background=[255,10,20], got %v", params.Background)
	}
	if params.Interlace == nil || *params.Interlace != true {
		t.Errorf("Expected Interlace=true")
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

func TestParseBool(t *testing.T) {
	t.Run("parseBool", func(t *testing.T) {
		if r, err := parseBool("true"); r != true {
			t.Errorf("Expected string true to result a native type true %s", err)
		}

		if r, err := parseBool("false"); r != false {
			t.Errorf("Expected string false to result a native type false %s", err)
		}

		if _, err := parseBool(""); err != nil {
			t.Errorf("Expected blank values to default to false, it didn't! %s", err)
		}

		if r, err := parseBool("foo"); err == nil {
			t.Errorf("Expected malformed values to result in an error, it didn't! %+v", r)
		}
	})
}

func TestGravityParam(t *testing.T) {
	params, _ := BuildParamsFromQuery(map[string][]string{"gravity": {"smart"}})
	if params.Gravity != bimg.GravitySmart {
		t.Errorf("Expected GravitySmart for 'smart', got %d", params.Gravity)
	}

	params2, _ := BuildParamsFromQuery(map[string][]string{"gravity": {"foo"}})
	if params2.Gravity != bimg.GravityCentre {
		t.Errorf("Expected GravityCentre for 'foo', got %d", params2.Gravity)
	}
}

func TestApplyQueryParam(t *testing.T) {
	t.Run("integer fields", func(t *testing.T) {
		cases := []struct {
			key   string
			value string
			want  int
		}{
			{"width", "100", 100},
			{"height", "200", 200},
			{"quality", "95", 95},
			{"rotate", "90", 90},
			{"speed", "5", 5},
		}

		for _, tc := range cases {
			var opts img.ImageOptions
			err := applyQueryParam(&opts, tc.key, tc.value)
			if err != nil {
				t.Errorf("applyQueryParam(%q, %q) returned error: %s", tc.key, tc.value, err)
			}
		}
	})

	t.Run("bool pointer fields", func(t *testing.T) {
		var opts img.ImageOptions
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
		var opts img.ImageOptions
		err := applyQueryParam(&opts, "palette", "false")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if opts.Palette == nil || *opts.Palette != false {
			t.Errorf("Expected Palette = *false (bug fix), got %v", opts.Palette)
		}
	})

	t.Run("unknown key is ignored", func(t *testing.T) {
		var opts img.ImageOptions
		err := applyQueryParam(&opts, "unknownparam", "value")
		if err != nil {
			t.Errorf("Expected unknown key to be ignored, got error: %s", err)
		}
	})

	t.Run("parsed string fields", func(t *testing.T) {
		var opts img.ImageOptions
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
