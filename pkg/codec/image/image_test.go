package image

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func ensureSampleJPEG(t *testing.T) {
	t.Helper()
	rootSample := filepath.Join("..", "..", "..", "testdata", "sample.jpg")
	if _, err := os.Stat(rootSample); err == nil {
		return
	}
	b64Path := filepath.Join("..", "..", "..", "testdata", "sample.jpg.b64")
	encoded, err := os.ReadFile(b64Path)
	if err != nil {
		t.Fatalf("read sample base64: %v", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		t.Fatalf("decode sample base64: %v", err)
	}
	if err := os.WriteFile(rootSample, decoded, 0o644); err != nil {
		t.Fatalf("write sample jpg: %v", err)
	}
}

func TestTranscodeToWebPSuccess(t *testing.T) {
	t.Parallel()

	ensureSampleJPEG(t)
	in := filepath.Join("..", "..", "..", "testdata", "sample.jpg")
	out := filepath.Join("..", "..", "..", "build", "test-sample.webp")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	_ = os.Remove(out)

	tr := NewTranscoder()
	if err := tr.TranscodeToWebP(context.Background(), in, out, 82); err != nil {
		t.Fatalf("transcode: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if len(data) < 32 {
		t.Fatalf("output too small: %d bytes", len(data))
	}
	if string(data[0:4]) != "RIFF" {
		t.Fatalf("missing RIFF header: %q", data[0:4])
	}
	if !containsWEBP(data[:32]) {
		t.Fatalf("missing WEBP marker in first 32 bytes")
	}
}

func TestTranscodeToWebPMissingInputFails(t *testing.T) {
	t.Parallel()

	tr := NewTranscoder()
	err := tr.TranscodeToWebP(context.Background(), "./does-not-exist.jpg", filepath.Join("..", "..", "..", "build", "missing.webp"), 82)
	if err == nil {
		t.Fatal("expected error for missing input")
	}
}

func containsWEBP(b []byte) bool {
	for i := 0; i+4 <= len(b); i++ {
		if string(b[i:i+4]) == "WEBP" {
			return true
		}
	}
	return false
}
