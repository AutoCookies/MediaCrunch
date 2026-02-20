package sqlite

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"mediacrunch/internal/jobs"
)

func TestSQLiteStoreLifecycleAndMigrations(t *testing.T) {
	db := filepath.Join(t.TempDir(), "test.db")
	st, _ := New(db)
	ctx := context.Background()
	if err := st.Init(ctx); err != nil {
		t.Fatal(err)
	}
	j, err := st.CreateJob(ctx, jobs.Job{ID: "1", InputPath: "in.jpg", OutputPath: "out.webp", Codec: jobs.CodecWebP, Quality: 80, State: jobs.StateQueued})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateJobState(ctx, j.ID, jobs.StateRunning, nil); err != nil {
		t.Fatal(err)
	}
	if err := st.SetJobOutcome(ctx, j.ID, jobs.StateSuccess, "", 1, 1, 1, nil); err != nil {
		t.Fatal(err)
	}
	stats, err := st.Stats(ctx)
	if err != nil || stats.SuccessJobs != 1 {
		t.Fatalf("stats: %+v err=%v", stats, err)
	}

	out, err := exec.Command("sqlite3", db, `PRAGMA index_list('file_hashes');`).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "idx_file_hashes_fingerprint") || !strings.Contains(text, "idx_file_hashes_sha256") {
		t.Fatalf("indexes missing: %s", text)
	}
}
