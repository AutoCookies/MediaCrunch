package orchestrator

import (
	"context"
	"encoding/base64"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mediacrunch/internal/dedupe"
	"mediacrunch/internal/jobs"
	"mediacrunch/internal/metrics"
	"mediacrunch/internal/skip"
	storesqlite "mediacrunch/internal/store/sqlite"
	"mediacrunch/pkg/codec/image"
)

func ensureSample(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "..")
	enc, err := os.ReadFile(filepath.Join(root, "testdata", "sample.jpg.b64"))
	if err != nil {
		t.Fatal(err)
	}
	dec, _ := base64.StdEncoding.DecodeString(string(enc))
	in := filepath.Join(t.TempDir(), "sample.jpg")
	if err := os.WriteFile(in, dec, 0o644); err != nil {
		t.Fatal(err)
	}
	return in
}

func newOrch(t *testing.T, codec jobs.ImageCodec, outDir string) (*Orchestrator, *storesqlite.Store, context.Context, context.CancelFunc) {
	t.Helper()
	db := filepath.Join(t.TempDir(), "mc.db")
	st, _ := storesqlite.New(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := st.Init(ctx); err != nil {
		t.Fatal(err)
	}
	q := jobs.NewQueue(10)
	h := dedupe.NewHasher(1 << 20)
	idx := dedupe.NewSQLiteIndex(db)
	sp := skip.NewPolicy(st, idx, h, codec, 82)
	orch := New(q, st, image.NewTranscoder(), metrics.New(), slog.Default(), 1, codec, 82, outDir, sp, idx, h)
	go orch.Start(ctx)
	return orch, st, ctx, cancel
}

func TestSmartSkipOutputFresh(t *testing.T) {
	in := ensureSample(t)
	outDir := t.TempDir()
	orch, st, ctx, cancel := newOrch(t, jobs.CodecWebP, outDir)
	defer cancel()
	defer st.Close()
	if err := orch.Submit(ctx, in); err != nil {
		t.Fatal(err)
	}
	waitState(t, ctx, st, in, jobs.StateSuccess)
	if err := orch.Submit(ctx, in); err != nil {
		t.Fatal(err)
	}
	j := waitState(t, ctx, st, in, jobs.StateSkipped)
	if j.SkipReason != "output_fresh" {
		t.Fatalf("unexpected reason %q", j.SkipReason)
	}
}

func TestDedupeSkipHash(t *testing.T) {
	in := ensureSample(t)
	dup := filepath.Join(t.TempDir(), "dup.jpg")
	b, _ := os.ReadFile(in)
	_ = os.WriteFile(dup, b, 0o644)
	outDir := t.TempDir()
	orch, st, ctx, cancel := newOrch(t, jobs.CodecWebP, outDir)
	defer cancel()
	defer st.Close()
	if err := orch.Submit(ctx, in); err != nil {
		t.Fatal(err)
	}
	waitState(t, ctx, st, in, jobs.StateSuccess)
	if err := orch.Submit(ctx, dup); err != nil {
		t.Fatal(err)
	}
	j := waitState(t, ctx, st, dup, jobs.StateSkipped)
	if j.SkipReason != "deduped_hash" {
		t.Fatalf("unexpected reason %q", j.SkipReason)
	}
}

func waitState(t *testing.T, ctx context.Context, st *storesqlite.Store, in string, s jobs.JobState) jobs.Job {
	t.Helper()
	d := time.Now().Add(4 * time.Second)
	for time.Now().Before(d) {
		j, err := st.FindLatestByInputPath(ctx, in)
		if err == nil && j != nil && j.State == s {
			return *j
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("state %s not reached", s)
	return jobs.Job{}
}
