//go:build cgo && ffmpeg

package metadata

import (
	"encoding/json"
	"strings"

	"github.com/asticode/go-astiav"
	"github.com/h2non/filetype"

	"github.com/h2non/imaginary/internal/avio"
	img "github.com/h2non/imaginary/internal/image"
)

const durationTimeBase = float64(astiav.TimeBase) // AV_TIME_BASE = 1,000,000

// ExtractAV extracts metadata from an audio or video buffer using FFmpeg via go-astiav.
// It creates a custom AVIO context to read from the []byte buffer without
// writing to a temporary file.
func ExtractAV(buf []byte) (MetadataResult, error) {
	formatCtx := astiav.AllocFormatContext()
	if formatCtx == nil {
		return MetadataResult{}, img.New(img.KindProcessing, "Cannot allocate format context")
	}
	defer formatCtx.Free()

	ioCtx, err := avio.NewMemoryIOContext(buf)
	if err != nil {
		return MetadataResult{}, img.Wrap(img.KindProcessing, "Cannot create IO context", err)
	}
	defer ioCtx.Free()

	formatCtx.SetPb(ioCtx)

	if err := formatCtx.OpenInput("", nil, nil); err != nil {
		return MetadataResult{}, img.Wrap(img.KindProcessing, "Cannot open media", err)
	}
	defer formatCtx.CloseInput()

	if err := formatCtx.FindStreamInfo(nil); err != nil {
		return MetadataResult{}, img.Wrap(img.KindProcessing, "Cannot find stream info", err)
	}

	result, err := buildAVMetadata(formatCtx, buf)
	if err != nil {
		return MetadataResult{}, err
	}

	body, err := json.Marshal(result)
	if err != nil {
		return MetadataResult{}, img.Wrap(img.KindProcessing, "Cannot serialize media metadata", err)
	}

	return MetadataResult{Body: body, Mime: "application/json"}, nil
}

// buildAVMetadata classifies the probed streams and constructs audio or video metadata.
func buildAVMetadata(fc *astiav.FormatContext, buf []byte) (Metadata, error) {
	format := ""
	if ifmt := fc.InputFormat(); ifmt != nil {
		format = ifmt.Name()
	}

	duration := mediaDuration(fc)
	hasVideo := false
	hasAudio := false
	streams := make([]StreamMetadata, 0, len(fc.Streams()))

	for _, stream := range fc.Streams() {
		switch stream.CodecParameters().MediaType() {
		case astiav.MediaTypeVideo:
			if isVisualVideoStream(stream) {
				hasVideo = true
			}
		case astiav.MediaTypeAudio:
			hasAudio = true
		}
		streams = append(streams, buildStreamMetadata(stream))
	}

	if hasVideo {
		width, height := primaryVideoDimensions(fc)
		videoMeta := VideoMetadata{
			Format:   format,
			Duration: duration,
			Size:     int64(len(buf)),
			Width:    width,
			Height:   height,
			BitRate:  fc.BitRate(),
			Streams:  streams,
		}
		if md := fc.Metadata(); md != nil {
			videoMeta.Tags = dictToMap(md)
		}

		return Metadata{
			MediaType: "video",
			SizeBytes: int64(len(buf)),
			MIMEType:  avMIMEType(buf, "video", format),
			Video:     &videoMeta,
		}, nil
	}

	if hasAudio {
		return Metadata{
			MediaType: "audio",
			SizeBytes: int64(len(buf)),
			MIMEType:  avMIMEType(buf, "audio", format),
			Audio: &AudioMetadata{
				Format:   format,
				Duration: duration,
			},
		}, nil
	}

	return Metadata{}, img.New(img.KindUnsupportedMedia, "No audio or video stream found")
}

func mediaDuration(fc *astiav.FormatContext) float64 {
	if dur := fc.Duration(); dur > 0 {
		return float64(dur) / durationTimeBase
	}

	var duration float64
	for _, stream := range fc.Streams() {
		dur := stream.Duration()
		timeBase := stream.TimeBase()
		if dur > 0 && timeBase.Den() > 0 {
			streamDuration := float64(dur) * timeBase.Float64()
			if streamDuration > duration {
				duration = streamDuration
			}
		}
	}
	return duration
}

func primaryVideoDimensions(fc *astiav.FormatContext) (int, int) {
	if stream, _, err := fc.FindBestStream(astiav.MediaTypeVideo, -1, -1); err == nil {
		if isVisualVideoStream(stream) {
			params := stream.CodecParameters()
			return params.Width(), params.Height()
		}
	}

	for _, stream := range fc.Streams() {
		if isVisualVideoStream(stream) {
			params := stream.CodecParameters()
			return params.Width(), params.Height()
		}
	}
	return 0, 0
}

func isVisualVideoStream(stream *astiav.Stream) bool {
	if stream.CodecParameters().MediaType() != astiav.MediaTypeVideo {
		return false
	}
	flags := stream.DispositionFlags()
	return !flags.Has(astiav.DispositionFlagAttachedPic) &&
		!flags.Has(astiav.DispositionFlagStillImage) &&
		!flags.Has(astiav.DispositionFlagTimedThumbnails)
}

func avMIMEType(buf []byte, mediaType, format string) string {
	kind, err := filetype.Get(buf)
	if err == nil && kind.MIME.Value != "" {
		mimeType := kind.MIME.Value
		if mediaType == "audio" && strings.HasPrefix(mimeType, "video/") {
			if strings.Contains(format, "mp4") || strings.Contains(format, "mov") {
				return "audio/mp4"
			}
			return "audio/" + strings.TrimPrefix(mimeType, "video/")
		}
		if mediaType == "video" && strings.HasPrefix(mimeType, "audio/") {
			if strings.Contains(format, "mp4") || strings.Contains(format, "mov") {
				return "video/mp4"
			}
			return "video/" + strings.TrimPrefix(mimeType, "audio/")
		}
		return mimeType
	}

	return "application/octet-stream"
}

// buildStreamMetadata extracts metadata from a single stream.
func buildStreamMetadata(s *astiav.Stream) StreamMetadata {
	cp := s.CodecParameters()

	sm := StreamMetadata{
		Index:  s.Index(),
		Codec:  cp.CodecID().Name(),
		Width:  cp.Width(),
		Height: cp.Height(),
	}

	sm.Type = cp.MediaType().String()

	if cp.BitRate() > 0 {
		sm.BitRate = cp.BitRate()
	}

	if dur := s.Duration(); dur > 0 {
		tb := s.TimeBase()
		if tb.Den() > 0 {
			sm.Duration = float64(dur) * tb.Float64()
		}
	}

	if afr := s.AvgFrameRate(); afr.Den() > 0 {
		sm.FrameRate = afr.String()
	}

	if cp.SampleRate() > 0 {
		sm.SampleRate = cp.SampleRate()
	}
	if cl := cp.ChannelLayout(); cl.Valid() {
		sm.Channels = cl.Channels()
	}

	if md := s.Metadata(); md != nil {
		sm.Tags = dictToMap(md)
	}

	return sm
}

// dictToMap converts an FFmpeg Dictionary to a Go map.
func dictToMap(d *astiav.Dictionary) map[string]string {
	if d == nil {
		return nil
	}

	m := make(map[string]string)
	var prev *astiav.DictionaryEntry

	for {
		entry := d.Get("", prev, astiav.DictionaryFlags(astiav.DictionaryFlagIgnoreSuffix))
		if entry == nil {
			break
		}
		m[entry.Key()] = entry.Value()
		prev = entry
	}

	if len(m) == 0 {
		return nil
	}

	return m
}
