package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mediacrunch/internal/jobs"
	"mediacrunch/internal/metrics"
	"mediacrunch/internal/output"
	"mediacrunch/internal/store"
)

type ImageTranscoder interface {
	TranscodeToWebP(ctx context.Context, inPath, outPath string, quality int) error
}

type Orchestrator struct {
	queue      *jobs.Queue
	store      store.JobStore
	transcoder ImageTranscoder
	metrics    *metrics.Metrics
	logger     *slog.Logger
	workers    int
	quality    int
	outputDir  string
	counter    atomic.Int64
}

func New(queue *jobs.Queue, st store.JobStore, transcoder ImageTranscoder, m *metrics.Metrics, logger *slog.Logger, workers, quality int, outputDir string) *Orchestrator {
	if workers <= 0 {
		workers = 1
	}
	return &Orchestrator{queue: queue, store: st, transcoder: transcoder, metrics: m, logger: logger, workers: workers, quality: quality, outputDir: outputDir}
}

func (o *Orchestrator) Start(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < o.workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			o.worker(ctx, id)
		}(i)
	}
	<-ctx.Done()
	wg.Wait()
}

func (o *Orchestrator) Submit(ctx context.Context, inputPath string) error {
	jobID := fmt.Sprintf("job-%d-%d", time.Now().UnixNano(), o.counter.Add(1))
	outPath := filepath.Join(o.outputDir, strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))+".webp")
	j := jobs.Job{ID: jobID, InputPath: inputPath, OutputPath: outPath, Quality: o.quality, State: jobs.StateQueued}
	if _, err := o.store.CreateJob(ctx, j); err != nil {
		return fmt.Errorf("create queued job: %w", err)
	}
	if err := o.queue.Enqueue(jobID); err != nil {
		o.metrics.IncRejected()
		o.logger.Error("queue rejection", "job_id", jobID, "path", inputPath, "error", err)
		msg := err.Error()
		_ = o.store.UpdateJobState(ctx, jobID, jobs.StateRunning, nil)
		_ = o.store.UpdateJobState(ctx, jobID, jobs.StateFailed, &msg)
		return err
	}
	o.metrics.IncEnqueued()
	o.logger.Info("job enqueued", "job_id", jobID, "input", inputPath, "output", outPath)
	return nil
}

func (o *Orchestrator) worker(ctx context.Context, wid int) {
	for {
		select {
		case <-ctx.Done():
			return
		case id, ok := <-o.queue.Dequeue():
			if !ok {
				return
			}
			o.process(ctx, wid, id)
		}
	}
}

func (o *Orchestrator) process(ctx context.Context, wid int, id string) {
	j, err := o.store.GetJob(ctx, id)
	if err != nil {
		o.logger.Error("load job failed", "job_id", id, "error", err)
		return
	}
	if err := o.store.UpdateJobState(ctx, id, jobs.StateRunning, nil); err != nil {
		o.logger.Error("transition queued->running failed", "job_id", id, "error", err)
		return
	}
	o.logger.Info("job running", "job_id", id, "worker", wid)

	skip, skipReason := o.shouldSkip(ctx, j)
	if skip {
		_ = o.store.UpdateJobState(ctx, id, jobs.StateSkipped, nil)
		o.metrics.IncSkipped()
		o.logger.Info("job skipped", "job_id", id, "reason", skipReason)
		return
	}

	if err := os.MkdirAll(filepath.Dir(j.OutputPath), 0o755); err != nil {
		o.failJob(ctx, id, fmt.Errorf("create output dir: %w", err))
		return
	}
	tmp := output.TempPath(j.OutputPath)
	defer os.Remove(tmp)

	start := time.Now()
	if err := o.transcoder.TranscodeToWebP(ctx, j.InputPath, tmp, j.Quality); err != nil {
		o.failJob(ctx, id, err)
		return
	}
	if err := output.CommitTempFile(tmp, j.OutputPath); err != nil {
		o.failJob(ctx, id, err)
		return
	}
	inSize, outSize := fileSize(j.InputPath), fileSize(j.OutputPath)
	o.metrics.AddBytes(inSize, outSize)
	o.metrics.IncProcessed()
	if err := o.store.UpdateJobState(ctx, id, jobs.StateSuccess, nil); err != nil {
		o.logger.Error("transition running->success failed", "job_id", id, "error", err)
		return
	}
	o.logger.Info("job success", "job_id", id, "duration_ms", time.Since(start).Milliseconds(), "bytes_in", inSize, "bytes_out", outSize)
}

func (o *Orchestrator) shouldSkip(ctx context.Context, j jobs.Job) (bool, string) {
	if validAndNewer(j.InputPath, j.OutputPath) {
		return true, "output_exists_newer"
	}
	latest, err := o.store.FindLatestByInputPath(ctx, j.InputPath)
	if err == nil && latest != nil && latest.State == jobs.StateSuccess && validWebP(latest.OutputPath) {
		return true, "latest_success_in_store"
	}
	return false, ""
}

func validAndNewer(inputPath, outputPath string) bool {
	in, err := os.Stat(inputPath)
	if err != nil {
		return false
	}
	out, err := os.Stat(outputPath)
	if err != nil {
		return false
	}
	return !out.ModTime().Before(in.ModTime()) && validWebP(outputPath)
}

func validWebP(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil || len(b) < 12 {
		return false
	}
	return string(b[:4]) == "RIFF" && strings.Contains(string(b[:32]), "WEBP")
}

func (o *Orchestrator) failJob(ctx context.Context, id string, err error) {
	msg := err.Error()
	o.metrics.IncFailed()
	_ = o.store.UpdateJobState(ctx, id, jobs.StateFailed, &msg)
	o.logger.Error("job failed", "job_id", id, "error", err)
}

func fileSize(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.Size()
}
