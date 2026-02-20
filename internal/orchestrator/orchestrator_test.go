package orchestrator

import (
	"context"
	"encoding/base64"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mediacrunch/internal/jobs"
	"mediacrunch/internal/metrics"
	storesqlite "mediacrunch/internal/store/sqlite"
	"mediacrunch/pkg/codec/image"
)

func ensureSample(t *testing.T, root string) string {
	t.Helper()
	b64Path := filepath.Join(root, "testdata", "sample.jpg.b64")
	enc, err := os.ReadFile(b64Path)
	if err != nil {
		t.Fatalf("read b64: %v", err)
	}
	dec, err := base64.StdEncoding.DecodeString(string(enc))
	if err != nil {
		t.Fatalf("decode b64: %v", err)
	}
	inDir := filepath.Join(t.TempDir(), "in")
	if err := os.MkdirAll(inDir, 0o755); err != nil {
		t.Fatal(err)
	}
	in := filepath.Join(inDir, "sample.jpg")
	if err := os.WriteFile(in, dec, 0o644); err != nil {
		t.Fatal(err)
	}
	return in
}

func TestSkipPolicyOutputNewer(t *testing.T) {
	in := filepath.Join(t.TempDir(), "a.jpg")
	out := filepath.Join(t.TempDir(), "a.webp")
	if err := os.WriteFile(in, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(out, []byte("RIFFxxxxWEBPxxxx"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !validAndNewer(in, out) {
		t.Fatal("expected skip condition")
	}
}

func TestOrchestratorIntegration(t *testing.T) {
	root := filepath.Join("..", "..")
	in := ensureSample(t, root)
	outDir := filepath.Join(t.TempDir(), "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := storesqlite.New(filepath.Join(t.TempDir(), "mc.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err := st.Init(ctx); err != nil {
		t.Fatal(err)
	}
	q := jobs.NewQueue(10)
	m := metrics.New()
	orch := New(q, st, image.NewTranscoder(), m, slog.Default(), 1, 82, outDir)
	go orch.Start(ctx)
	if err := orch.Submit(ctx, in); err != nil {
		t.Fatalf("submit: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		latest, err := st.FindLatestByInputPath(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if latest != nil && latest.State == jobs.StateSuccess {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("job did not reach success, latest=%+v", latest)
		}
		time.Sleep(20 * time.Millisecond)
	}
	out := filepath.Join(outDir, "sample.webp")
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if len(b) < 32 || string(b[:4]) != "RIFF" {
		t.Fatalf("invalid webp header")
	}
}
