package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTranscodeValidateRejectsCodecMismatch(t *testing.T) {
	in := filepath.Join(t.TempDir(), "a.jpg")
	_ = os.WriteFile(in, []byte("x"), 0o644)
	cfg := TranscodeConfig{InPath: in, OutPath: filepath.Join(t.TempDir(), "a.webp"), Quality: 80, Codec: "avif"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected extension mismatch")
	}
}

func TestDaemonValidateRejectsInvalidWorkers(t *testing.T) {
	cfg := DaemonConfig{InputDir: t.TempDir(), OutputDir: t.TempDir(), DBPath: filepath.Join(t.TempDir(), "x.db"), Workers: 0, QueueSize: 1, Quality: 80, Codec: "webp", WriteMode: "atomic", OnSuccess: "keep"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}
