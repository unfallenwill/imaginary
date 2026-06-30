//go:build cgo && ffmpeg

package server

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/h2non/imaginary/internal/metadata"
)

func TestMetadataAudioPOST(t *testing.T) {
	buf := syntheticAudioWAV(8000, 100)
	req := httptest.NewRequest(http.MethodPost, "/metadata", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "audio/wav")
	rec := httptest.NewRecorder()

	NewServerMux(testMetadataConfig()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var meta metadata.Metadata
	if err := json.Unmarshal(rec.Body.Bytes(), &meta); err != nil {
		t.Fatalf("Cannot unmarshal response: %v", err)
	}
	if meta.MediaType != "audio" || meta.Audio == nil {
		t.Fatalf("Expected audio metadata, got %+v", meta)
	}
	if meta.SizeBytes != int64(len(buf)) {
		t.Errorf("Expected sizeBytes=%d, got %d", len(buf), meta.SizeBytes)
	}
	if meta.Audio.Duration < 0.09 || meta.Audio.Duration > 0.11 {
		t.Errorf("Expected duration close to 0.1 seconds, got %f", meta.Audio.Duration)
	}
}

func syntheticAudioWAV(sampleRate, durationMillis int) []byte {
	sampleCount := sampleRate * durationMillis / 1000
	dataSize := sampleCount * 2

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
