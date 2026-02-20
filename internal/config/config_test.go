package config

import "testing"

func TestValidateRejectsLowQuality(t *testing.T) {
	cfg := TranscodeConfig{InPath: "testdata/sample.jpg", OutPath: "build/out.webp", Quality: 0}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateRejectsHighQuality(t *testing.T) {
	cfg := TranscodeConfig{InPath: "testdata/sample.jpg", OutPath: "build/out.webp", Quality: 101}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateRejectsMissingInput(t *testing.T) {
	cfg := TranscodeConfig{InPath: "testdata/missing.jpg", OutPath: "build/out.webp", Quality: 80}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateRejectsUnsupportedExtension(t *testing.T) {
	cfg := TranscodeConfig{InPath: "testdata/sample.gif", OutPath: "build/out.webp", Quality: 80}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestDaemonValidateRejectsInvalidWorkers(t *testing.T) {
	cfg := DaemonConfig{InputDir: t.TempDir(), OutputDir: t.TempDir(), DBPath: t.TempDir() + "/x.db", Workers: 0, QueueSize: 1, Quality: 80, WriteMode: "atomic", OnSuccess: "keep"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}
