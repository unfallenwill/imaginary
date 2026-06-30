//go:build cgo && ffmpeg

package metadata

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/gif"
	"testing"

	"github.com/asticode/go-astiav"
)

func TestExtractAudio(t *testing.T) {
	buf := syntheticWAV(8000, 100)

	result, err := Extract(buf, MediaTypeAudio)
	if err != nil {
		t.Fatalf("Extract audio failed: %v", err)
	}

	var meta Metadata
	if err := json.Unmarshal(result.Body, &meta); err != nil {
		t.Fatalf("Cannot unmarshal audio metadata: %v", err)
	}

	if meta.MediaType != "audio" {
		t.Fatalf("Expected mediaType=audio, got %s", meta.MediaType)
	}
	if meta.Audio == nil {
		t.Fatal("Audio metadata is nil")
	}
	if meta.Video != nil || meta.Image != nil {
		t.Fatal("Non-audio metadata should be nil")
	}
	if meta.SizeBytes != int64(len(buf)) {
		t.Errorf("Expected sizeBytes=%d, got %d", len(buf), meta.SizeBytes)
	}
	if meta.MIMEType != "audio/x-wav" {
		t.Errorf("Expected mimeType=audio/x-wav, got %s", meta.MIMEType)
	}
	if meta.Audio.Duration < 0.09 || meta.Audio.Duration > 0.11 {
		t.Errorf("Expected duration close to 0.1 seconds, got %f", meta.Audio.Duration)
	}
}

func TestExtractVideoDimensions(t *testing.T) {
	buf := syntheticAnimatedGIF(t, 2, 3)

	result, err := ExtractAV(buf)
	if err != nil {
		t.Fatalf("Extract video failed: %v", err)
	}

	var meta Metadata
	if err := json.Unmarshal(result.Body, &meta); err != nil {
		t.Fatalf("Cannot unmarshal video metadata: %v", err)
	}

	if meta.MediaType != "video" {
		t.Fatalf("Expected mediaType=video, got %s", meta.MediaType)
	}
	if meta.Video == nil {
		t.Fatal("Video metadata is nil")
	}
	if meta.Video.Width != 2 || meta.Video.Height != 3 {
		t.Errorf("Expected dimensions 2x3, got %dx%d", meta.Video.Width, meta.Video.Height)
	}
	if meta.SizeBytes != int64(len(buf)) {
		t.Errorf("Expected sizeBytes=%d, got %d", len(buf), meta.SizeBytes)
	}
}

func TestIsVisualVideoStreamIgnoresArtwork(t *testing.T) {
	formatCtx := astiav.AllocFormatContext()
	if formatCtx == nil {
		t.Fatal("Cannot allocate format context")
	}
	defer formatCtx.Free()

	stream := formatCtx.NewStream(nil)
	stream.CodecParameters().SetMediaType(astiav.MediaTypeVideo)
	if !isVisualVideoStream(stream) {
		t.Fatal("Regular video stream should be visual")
	}

	stream.SetDispositionFlags(astiav.NewDispositionFlags(astiav.DispositionFlagAttachedPic))
	if isVisualVideoStream(stream) {
		t.Fatal("Attached artwork should not classify audio as video")
	}
}

func TestAVMIMETypeNormalizesAudioOnlyMP4(t *testing.T) {
	buf := []byte{0, 0, 0, 12, 'f', 't', 'y', 'p', 'm', 'p', '4', '2'}
	if got := avMIMEType(buf, "audio", "mov,mp4,m4a,3gp,3g2,mj2"); got != "audio/mp4" {
		t.Fatalf("Expected audio/mp4, got %s", got)
	}
}

func syntheticWAV(sampleRate, durationMillis int) []byte {
	sampleCount := sampleRate * durationMillis / 1000
	dataSize := sampleCount * 2 // mono, signed 16-bit PCM

	var buf bytes.Buffer
	buf.WriteString("RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(sampleRate*2))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(16))
	buf.WriteString("data")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(dataSize))
	buf.Write(make([]byte, dataSize))
	return buf.Bytes()
}

func syntheticAnimatedGIF(t *testing.T, width, height int) []byte {
	t.Helper()

	palette := color.Palette{color.Black, color.White}
	first := image.NewPaletted(image.Rect(0, 0, width, height), palette)
	second := image.NewPaletted(image.Rect(0, 0, width, height), palette)
	for i := range second.Pix {
		second.Pix[i] = 1
	}

	var buf bytes.Buffer
	err := gif.EncodeAll(&buf, &gif.GIF{
		Image: []*image.Paletted{first, second},
		Delay: []int{5, 5},
	})
	if err != nil {
		t.Fatalf("Cannot create test GIF: %v", err)
	}
	return buf.Bytes()
}
