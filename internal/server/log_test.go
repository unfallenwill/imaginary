package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/h2non/imaginary/internal/server"
)

type fakeWriter func([]byte) (int, error)

func (fake fakeWriter) Write(buf []byte) (int, error) {
	return fake(buf)
}

func TestLogInfo(t *testing.T) {
	var buf []byte
	writer := fakeWriter(func(b []byte) (int, error) {
		buf = b
		return len(b), nil
	})

	noopHandler := func(w http.ResponseWriter, r *http.Request) {}
	log := server.NewLog(http.HandlerFunc(noopHandler), writer, "info")

	ts := httptest.NewServer(log)
	defer ts.Close()

	_, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	if len(buf) == 0 {
		t.Fatal("expected log output, got empty")
	}

	var entry map[string]interface{}
	if err := json.Unmarshal(buf, &entry); err != nil {
		t.Fatalf("expected valid JSON log output: %v", err)
	}

	if entry["method"] != "GET" {
		t.Fatalf("expected method GET, got %v", entry["method"])
	}
	if entry["status"] != float64(200) {
		t.Fatalf("expected status 200, got %v", entry["status"])
	}
	if entry["path"] == "" {
		t.Fatal("expected non-empty path")
	}
	if entry["ip"] == "" {
		t.Fatal("expected non-empty ip")
	}
	if entry["duration"] == nil {
		t.Fatal("expected duration field")
	}
	if entry["bytes"] == nil {
		t.Fatal("expected bytes field")
	}
}

func TestLogError(t *testing.T) {
	var buf []byte
	writer := fakeWriter(func(b []byte) (int, error) {
		buf = b
		return len(b), nil
	})

	noopHandler := func(w http.ResponseWriter, r *http.Request) {}
	log := server.NewLog(http.HandlerFunc(noopHandler), writer, "error")

	ts := httptest.NewServer(log)
	defer ts.Close()

	_, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	data := string(buf)
	if data != "" {
		t.Fatalf("expected no log output for 200 OK with error level, got: %s", data)
	}
}

func TestLogWarn(t *testing.T) {
	var buf []byte
	writer := fakeWriter(func(b []byte) (int, error) {
		buf = b
		return len(b), nil
	})

	errorHandler := func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}
	log := server.NewLog(http.HandlerFunc(errorHandler), writer, "warn")

	ts := httptest.NewServer(log)
	defer ts.Close()

	_, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	if len(buf) == 0 {
		t.Fatal("expected log output for 400 with warn level, got empty")
	}

	var entry map[string]interface{}
	if err := json.Unmarshal(buf, &entry); err != nil {
		t.Fatalf("expected valid JSON log output: %v", err)
	}

	if entry["status"] != float64(400) {
		t.Fatalf("expected status 400, got %v", entry["status"])
	}
	if entry["level"] != "WARN" {
		t.Fatalf("expected level WARN, got %v", entry["level"])
	}
}
