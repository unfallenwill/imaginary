package metadata

import (
	"encoding/json"

	"github.com/h2non/bimg"

	img "github.com/h2non/imaginary/internal/image"
)

// MediaType classifies the input data for metadata extraction.
type MediaType int

const (
	MediaTypeImage MediaType = iota
	MediaTypeVideo
	MediaTypeUnknown
)

// Metadata holds the extracted metadata from any media type.
// Only one of Image or Video will be populated, based on the media type.
type Metadata struct {
	MediaType string         `json:"mediaType"`
	Image     *ImageMetadata `json:"image,omitempty"`
	Video     *VideoMetadata `json:"video,omitempty"`
}

// ImageMetadata contains metadata extracted from an image via bimg.
type ImageMetadata struct {
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Type        string    `json:"type"`
	Space       string    `json:"space"`
	Alpha       bool      `json:"hasAlpha"`
	Profile     bool      `json:"hasProfile"`
	Channels    int       `json:"channels"`
	Orientation int       `json:"orientation"`
	EXIF        bimg.EXIF `json:"exif"`
}

// VideoMetadata contains metadata extracted from a video via FFmpeg/libavformat.
type VideoMetadata struct {
	Format   string            `json:"format"`
	Duration float64           `json:"duration"`
	Size     int64             `json:"size"`
	BitRate  int64             `json:"bitRate"`
	Tags     map[string]string `json:"tags,omitempty"`
	Streams  []StreamMetadata  `json:"streams"`
}

// StreamMetadata describes a single stream (video/audio/subtitle) in a video.
type StreamMetadata struct {
	Index      int               `json:"index"`
	Type       string            `json:"streamType"`
	Codec      string            `json:"codec"`
	Width      int               `json:"width,omitempty"`
	Height     int               `json:"height,omitempty"`
	Duration   float64           `json:"duration,omitempty"`
	BitRate    int64             `json:"bitRate,omitempty"`
	FrameRate  string            `json:"frameRate,omitempty"`
	SampleRate int               `json:"sampleRate,omitempty"`
	Channels   int               `json:"channels,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
}

// MetadataResult wraps metadata extraction output as a JSON-serializable result.
type MetadataResult struct {
	Body []byte
	Mime string
}

// Extract dispatches metadata extraction based on the detected media type.
func Extract(buf []byte, mediaType MediaType) (MetadataResult, error) {
	switch mediaType {
	case MediaTypeImage:
		return ExtractImage(buf)
	case MediaTypeVideo:
		return ExtractVideo(buf)
	default:
		return MetadataResult{}, img.NewError("Unsupported media type for metadata extraction", img.KindUnsupportedMedia)
	}
}

// ExtractImage extracts metadata from an image buffer using bimg.
func ExtractImage(buf []byte) (MetadataResult, error) {
	meta, err := bimg.Metadata(buf)
	if err != nil {
		return MetadataResult{}, img.WrapError("Cannot retrieve image metadata", img.KindProcessing, err)
	}

	imageMeta := ImageMetadata{
		Width:       meta.Size.Width,
		Height:      meta.Size.Height,
		Type:        meta.Type,
		Space:       meta.Space,
		Alpha:       meta.Alpha,
		Profile:     meta.Profile,
		Channels:    meta.Channels,
		Orientation: meta.Orientation,
		EXIF:        meta.EXIF,
	}

	result := Metadata{
		MediaType: "image",
		Image:     &imageMeta,
	}

	body, err := json.Marshal(result)
	if err != nil {
		return MetadataResult{}, img.WrapError("Cannot serialize image metadata", img.KindProcessing, err)
	}

	return MetadataResult{Body: body, Mime: "application/json"}, nil
}
