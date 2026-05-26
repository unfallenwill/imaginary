//go:build cgo && ffmpeg

package metadata

import (
	"encoding/json"

	"github.com/asticode/go-astiav"

	"github.com/h2non/imaginary/internal/avio"
	img "github.com/h2non/imaginary/internal/image"
)

const durationTimeBase = float64(astiav.TimeBase) // AV_TIME_BASE = 1,000,000

// ExtractVideo extracts metadata from a video buffer using FFmpeg via go-astiav.
// It creates a custom AVIO context to read from the []byte buffer without
// writing to a temporary file.
func ExtractVideo(buf []byte) (MetadataResult, error) {
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
		return MetadataResult{}, img.Wrap(img.KindProcessing, "Cannot open video", err)
	}
	defer formatCtx.CloseInput()

	if err := formatCtx.FindStreamInfo(nil); err != nil {
		return MetadataResult{}, img.Wrap(img.KindProcessing, "Cannot find stream info", err)
	}

	result := buildVideoMetadata(formatCtx, len(buf))

	body, err := json.Marshal(result)
	if err != nil {
		return MetadataResult{}, img.Wrap(img.KindProcessing, "Cannot serialize video metadata", err)
	}

	return MetadataResult{Body: body, Mime: "application/json"}, nil
}

// buildVideoMetadata constructs VideoMetadata from a probed FormatContext.
func buildVideoMetadata(fc *astiav.FormatContext, fileSize int) Metadata {
	videoMeta := VideoMetadata{
		Size:    int64(fileSize),
		Streams: []StreamMetadata{},
	}

	if ifmt := fc.InputFormat(); ifmt != nil {
		videoMeta.Format = ifmt.Name()
	}

	if dur := fc.Duration(); dur > 0 {
		videoMeta.Duration = float64(dur) / durationTimeBase
	}

	videoMeta.BitRate = fc.BitRate()

	if md := fc.Metadata(); md != nil {
		videoMeta.Tags = dictToMap(md)
	}

	for _, stream := range fc.Streams() {
		sm := buildStreamMetadata(fc, stream)
		videoMeta.Streams = append(videoMeta.Streams, sm)
	}

	return Metadata{
		MediaType: "video",
		Video:     &videoMeta,
	}
}

// buildStreamMetadata extracts metadata from a single stream.
func buildStreamMetadata(fc *astiav.FormatContext, s *astiav.Stream) StreamMetadata {
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
