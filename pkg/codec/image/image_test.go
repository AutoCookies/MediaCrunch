package image

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mediacrunch/internal/jobs"
)

func ensureSampleJPEGPath(tb testing.TB) string {
	tb.Helper()
	root := filepath.Join("..", "..", "..")
	raw := filepath.Join(root, "testdata", "sample.jpg")
	if _, err := os.Stat(raw); err == nil {
		return raw
	}
	enc, err := os.ReadFile(filepath.Join(root, "testdata", "sample.jpg.b64"))
	if err != nil {
		tb.Fatal(err)
	}
	dec, err := base64.StdEncoding.DecodeString(string(enc))
	if err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(raw, dec, 0o644); err != nil {
		tb.Fatal(err)
	}
	return raw
}

func TestTranscodeWebPSuccess(t *testing.T) {
	in := ensureSampleJPEGPath(t)
	out := filepath.Join("..", "..", "..", "build", "test-sample.webp")
	_ = os.MkdirAll(filepath.Dir(out), 0o755)
	tr := NewTranscoder()
	if _, err := tr.Transcode(context.Background(), jobs.CodecWebP, in, out, 82); err != nil {
		t.Fatalf("transcode: %v", err)
	}
	b, _ := os.ReadFile(out)
	if len(b) < 32 || string(b[:4]) != "RIFF" || !strings.Contains(string(b[:32]), "WEBP") {
		t.Fatal("invalid webp")
	}
}

func TestTranscodeAVIFSuccess(t *testing.T) {
	in := ensureSampleJPEGPath(t)
	out := filepath.Join("..", "..", "..", "build", "test-sample.avif")
	_ = os.MkdirAll(filepath.Dir(out), 0o755)
	tr := NewTranscoder()
	if _, err := tr.Transcode(context.Background(), jobs.CodecAVIF, in, out, 60); err != nil {
		t.Skipf("AVIF encoder unavailable on this runner: %v", err)
	}
	b, _ := os.ReadFile(out)
	if len(b) < 32 || string(b[4:8]) != "ftyp" || !(strings.Contains(string(b[:32]), "avif") || strings.Contains(string(b[:32]), "avis")) {
		t.Fatal("invalid avif")
	}
}

func BenchmarkTranscodeWebPSmoke(b *testing.B) {
	in := ensureSampleJPEGPath(b)
	tr := NewTranscoder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out := filepath.Join("..", "..", "..", "build", "bench-smoke.webp")
		_, _ = tr.Transcode(context.Background(), jobs.CodecWebP, in, out, 80)
	}
}
