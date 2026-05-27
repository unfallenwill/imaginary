package image_test

import (
	"testing"

	"github.com/h2non/bimg"

	"github.com/h2non/imaginary/internal/image"
)

func TestBimgOptions(t *testing.T) {
	imgOpts := image.ImageOptions{
		Dimensions: image.Dimensions{Width: 500, Height: 600},
	}
	opts := image.BimgOptions(imgOpts)

	if opts.Width != imgOpts.Width || opts.Height != imgOpts.Height {
		t.Error("Invalid width and height")
	}
}

func TestBimgOptionsBackgroundMapping(t *testing.T) {
	imgOpts := image.ImageOptions{
		Transform: image.Transform{Background: []uint8{10, 20, 30}},
	}
	opts := image.BimgOptions(imgOpts)

	if opts.Background.R != 10 || opts.Background.G != 20 || opts.Background.B != 30 {
		t.Errorf("Background color mismatch: got R=%d G=%d B=%d, want 10,20,30",
			opts.Background.R, opts.Background.G, opts.Background.B)
	}
}

func TestBimgOptionsBackgroundEmpty(t *testing.T) {
	imgOpts := image.ImageOptions{}
	opts := image.BimgOptions(imgOpts)

	if opts.Background.R != 0 || opts.Background.G != 0 || opts.Background.B != 0 {
		t.Error("Empty background should result in zero bimg.Color")
	}
}

func TestBimgOptionsGaussianBlur(t *testing.T) {
	cases := []struct {
		name      string
		sigma     float64
		minAmpl   float64
		wantBlur  bool
		wantSigma float64
		wantAmpl  float64
	}{
		{"SigmaOnly", 5.0, 0, true, 5.0, 0},
		{"MinAmplOnly", 0, 0.3, true, 0, 0.3},
		{"BothSet", 3.5, 0.2, true, 3.5, 0.2},
		{"BothZero", 0, 0, false, 0, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			imgOpts := image.ImageOptions{
				Effects: image.Effects{Sigma: tc.sigma, MinAmpl: tc.minAmpl},
			}
			opts := image.BimgOptions(imgOpts)

			if tc.wantBlur {
				if opts.GaussianBlur.Sigma != tc.wantSigma {
					t.Errorf("Sigma: got %f, want %f", opts.GaussianBlur.Sigma, tc.wantSigma)
				}
				if opts.GaussianBlur.MinAmpl != tc.wantAmpl {
					t.Errorf("MinAmpl: got %f, want %f", opts.GaussianBlur.MinAmpl, tc.wantAmpl)
				}
			} else {
				if opts.GaussianBlur.Sigma != 0 || opts.GaussianBlur.MinAmpl != 0 {
					t.Error("GaussianBlur should be zero when both Sigma and MinAmpl are 0")
				}
			}
		})
	}
}

func TestBimgOptionsBoolFlags(t *testing.T) {
	t.Run("NilDefaultsToFalse", func(t *testing.T) {
		imgOpts := image.ImageOptions{}
		opts := image.BimgOptions(imgOpts)
		if opts.Flip || opts.Flop || opts.Force || opts.Embed || opts.StripMetadata {
			t.Error("Nil bool fields should default to false")
		}
	})

	t.Run("ExplicitTrue", func(t *testing.T) {
		imgOpts := image.ImageOptions{
			Flags: image.Flags{
				Flip:          boolPtr(true),
				StripMetadata: boolPtr(true),
				NoProfile:     boolPtr(true),
			},
		}
		opts := image.BimgOptions(imgOpts)
		if !opts.Flip || !opts.StripMetadata || !opts.NoProfile {
			t.Error("Explicit true bool fields should map to true")
		}
	})

	t.Run("ExplicitFalse", func(t *testing.T) {
		imgOpts := image.ImageOptions{
			Flags: image.Flags{
				Flip: boolPtr(false),
			},
		}
		opts := image.BimgOptions(imgOpts)
		if opts.Flip {
			t.Error("Explicit false bool field should map to false")
		}
	})
}

func TestBimgOptionsTypeMapping(t *testing.T) {
	cases := []struct {
		input string
		want  bimg.ImageType
	}{
		{"jpeg", bimg.JPEG},
		{"png", bimg.PNG},
		{"webp", bimg.WEBP},
		{"", bimg.UNKNOWN},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			imgOpts := image.ImageOptions{Type: tc.input}
			opts := image.BimgOptions(imgOpts)
			if opts.Type != tc.want {
				t.Errorf("Type: got %v, want %v", opts.Type, tc.want)
			}
		})
	}
}

func TestBimgOptionsTransformFields(t *testing.T) {
	imgOpts := image.ImageOptions{
		Transform: image.Transform{
			Rotate:     90,
			Extend:     bimg.ExtendWhite,
			Gravity:    bimg.GravitySmart,
			Colorspace: bimg.InterpretationBW,
		},
	}
	opts := image.BimgOptions(imgOpts)

	if opts.Rotate != bimg.Angle(90) {
		t.Errorf("Rotate: got %v, want %v", opts.Rotate, bimg.Angle(90))
	}
	if opts.Extend != bimg.ExtendWhite {
		t.Errorf("Extend: got %v, want %v", opts.Extend, bimg.ExtendWhite)
	}
	if opts.Gravity != bimg.GravitySmart {
		t.Errorf("Gravity: got %v, want %v", opts.Gravity, bimg.GravitySmart)
	}
	if opts.Interpretation != bimg.InterpretationBW {
		t.Errorf("Colorspace: got %v, want %v", opts.Interpretation, bimg.InterpretationBW)
	}
}

func TestBimgOptionsAspectRatioTransform(t *testing.T) {
	imgOpts := image.ImageOptions{
		Dimensions: image.Dimensions{Width: 1600},
		Transform:  image.Transform{AspectRatio: "16:9"},
	}
	opts := image.BimgOptions(imgOpts)

	if opts.Width != 1600 {
		t.Errorf("Width: got %d, want 1600", opts.Width)
	}
	if opts.Height != 900 {
		t.Errorf("Height: got %d, want 900 (1600 * 9 / 16)", opts.Height)
	}
}

func TestBimgOptionsAspectRatioIgnoredWhenBothSet(t *testing.T) {
	imgOpts := image.ImageOptions{
		Dimensions: image.Dimensions{Width: 100, Height: 200},
		Transform:  image.Transform{AspectRatio: "16:9"},
	}
	opts := image.BimgOptions(imgOpts)

	if opts.Width != 100 || opts.Height != 200 {
		t.Errorf("AspectRatio should be ignored: got %dx%d, want 100x200", opts.Width, opts.Height)
	}
}

// Helper to create a *bool for test readability.
func boolPtr(b bool) *bool {
	return &b
}
