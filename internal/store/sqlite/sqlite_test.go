package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"mediacrunch/internal/jobs"
)

func TestSQLiteStoreLifecycle(t *testing.T) {
	db := filepath.Join(t.TempDir(), "test.db")
	st, err := New(db)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer st.Close()
	ctx := context.Background()
	if err := st.Init(ctx); err != nil {
		t.Fatalf("init: %v", err)
	}
	j, err := st.CreateJob(ctx, jobs.Job{ID: "1", InputPath: "in.jpg", OutputPath: "out.webp", Quality: 80, State: jobs.StateQueued})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := st.UpdateJobState(ctx, j.ID, jobs.StateRunning, nil); err != nil {
		t.Fatalf("to running: %v", err)
	}
	if err := st.UpdateJobState(ctx, j.ID, jobs.StateSuccess, nil); err != nil {
		t.Fatalf("to success: %v", err)
	}
	stats, err := st.Stats(ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalJobs != 1 || stats.SuccessJobs != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}
