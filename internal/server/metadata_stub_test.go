//go:build !cgo || !ffmpeg

package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetadataAudioRequiresFFmpeg(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/metadata", bytes.NewReader([]byte{'I', 'D', '3', 4, 0, 0}))
	req.Header.Set("Content-Type", "audio/mpeg")
	rec := httptest.NewRecorder()

	NewServerMux(testMetadataConfig()).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("Expected 501, got %d: %s", rec.Code, rec.Body.String())
	}
}
