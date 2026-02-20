package store

import (
	"context"

	"mediacrunch/internal/jobs"
)

type JobStore interface {
	Init(ctx context.Context) error
	CreateJob(ctx context.Context, j jobs.Job) (jobs.Job, error)
	UpdateJobState(ctx context.Context, id string, state jobs.JobState, errMsg *string) error
	SetJobOutcome(ctx context.Context, id string, state jobs.JobState, skipReason string, bytesIn, bytesOut, durationMs int64, errMsg *string) error
	GetJob(ctx context.Context, id string) (jobs.Job, error)
	FindLatestByInputPath(ctx context.Context, inPath string) (*jobs.Job, error)
	Stats(ctx context.Context) (jobs.Stats, error)
	Close() error
}
