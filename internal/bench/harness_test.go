package bench

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"mediacrunch/internal/jobs"
	"mediacrunch/pkg/codec/image"
)

func TestHarnessWritesReport(t *testing.T) {
	inDir := filepath.Join(t.TempDir(), "in")
	out := filepath.Join(t.TempDir(), "out")
	_ = os.MkdirAll(inDir, 0o755)
	enc, _ := os.ReadFile(filepath.Join("..", "..", "testdata", "sample.jpg.b64"))
	dec, _ := base64.StdEncoding.DecodeString(string(enc))
	_ = os.WriteFile(filepath.Join(inDir, "a.jpg"), dec, 0o644)
	tr := image.NewTranscoder()
	rep, err := Run(context.Background(), jobs.CodecWebP, inDir, out, 82, 1, tr)
	if err != nil {
		t.Fatal(err)
	}
	if rep.FilesProcessed != 1 {
		t.Fatalf("unexpected rep: %+v", rep)
	}
	if _, err := os.Stat(filepath.Join(out, "report.json")); err != nil {
		t.Fatal(err)
	}
}
