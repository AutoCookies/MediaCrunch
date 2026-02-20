package jobs

import (
	"errors"
	"time"
)

type JobState string

type ImageCodec string

const (
	StateQueued  JobState = "queued"
	StateRunning JobState = "running"
	StateSuccess JobState = "success"
	StateFailed  JobState = "failed"
	StateSkipped JobState = "skipped"

	CodecWebP ImageCodec = "webp"
	CodecAVIF ImageCodec = "avif"
)

type Job struct {
	ID           string
	InputPath    string
	OutputPath   string
	Codec        ImageCodec
	Quality      int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	State        JobState
	SkipReason   string
	ErrorMessage *string
	BytesIn      int64
	BytesOut     int64
	DurationMs   int64
}

type Stats struct {
	TotalJobs     int64
	SuccessJobs   int64
	FailedJobs    int64
	SkippedJobs   int64
	TotalBytesIn  int64
	TotalBytesOut int64
}

var validTransitions = map[JobState]map[JobState]bool{
	StateQueued: {StateRunning: true},
	StateRunning: {
		StateSuccess: true,
		StateFailed:  true,
		StateSkipped: true,
	},
}

func ValidateTransition(from, to JobState) error {
	next, ok := validTransitions[from]
	if !ok || !next[to] {
		return errors.New("invalid job state transition")
	}
	return nil
}
