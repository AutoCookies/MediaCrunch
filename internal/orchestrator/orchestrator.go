package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mediacrunch/internal/dedupe"
	"mediacrunch/internal/jobs"
	"mediacrunch/internal/metrics"
	"mediacrunch/internal/output"
	"mediacrunch/internal/skip"
	"mediacrunch/internal/store"
	"mediacrunch/pkg/codec/image"
)

type ImageTranscoder interface {
	Transcode(ctx context.Context, codec jobs.ImageCodec, inPath, outPath string, quality int) (image.Result, error)
}

type Orchestrator struct {
	queue      *jobs.Queue
	store      store.JobStore
	transcoder ImageTranscoder
	metrics    *metrics.Metrics
	logger     *slog.Logger
	workers    int
	quality    int
	codec      jobs.ImageCodec
	outputDir  string
	skip       skip.SkipPolicy
	dedupe     dedupe.DedupeIndex
	hasher     dedupe.Hasher
	counter    atomic.Int64
}

func New(queue *jobs.Queue, st store.JobStore, transcoder ImageTranscoder, m *metrics.Metrics, logger *slog.Logger, workers int, codec jobs.ImageCodec, quality int, outputDir string, sp skip.SkipPolicy, dedupeIndex dedupe.DedupeIndex, hasher dedupe.Hasher) *Orchestrator {
	if workers <= 0 {
		workers = 1
	}
	return &Orchestrator{queue: queue, store: st, transcoder: transcoder, metrics: m, logger: logger, workers: workers, codec: codec, quality: quality, outputDir: outputDir, skip: sp, dedupe: dedupeIndex, hasher: hasher}
}

func (o *Orchestrator) Start(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < o.workers; i++ {
		wg.Add(1)
		go func(id int) { defer wg.Done(); o.worker(ctx, id) }(i)
	}
	<-ctx.Done()
	wg.Wait()
}

func (o *Orchestrator) Submit(ctx context.Context, inputPath string) error {
	jobID := fmt.Sprintf("job-%d-%d", time.Now().UnixNano(), o.counter.Add(1))
	outPath := filepath.Join(o.outputDir, strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))+extForCodec(o.codec))
	j := jobs.Job{ID: jobID, InputPath: inputPath, OutputPath: outPath, Codec: o.codec, Quality: o.quality, State: jobs.StateQueued}
	if _, err := o.store.CreateJob(ctx, j); err != nil {
		return err
	}
	if err := o.queue.Enqueue(jobID); err != nil {
		o.metrics.IncRejected()
		msg := err.Error()
		_ = o.store.UpdateJobState(ctx, jobID, jobs.StateRunning, nil)
		_ = o.store.SetJobOutcome(ctx, jobID, jobs.StateFailed, "queue_rejected", 0, 0, 0, &msg)
		return err
	}
	o.metrics.IncEnqueued()
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
		return
	}
	if err := o.store.UpdateJobState(ctx, id, jobs.StateRunning, nil); err != nil {
		return
	}
	skipIt, reason, err := o.skip.ShouldSkip(ctx, j.InputPath, j.OutputPath)
	if err == nil && skipIt {
		o.metrics.IncSkipped()
		_ = o.store.SetJobOutcome(ctx, id, jobs.StateSkipped, reason, 0, 0, 0, nil)
		o.logger.Info("job skipped", "job_id", id, "reason", reason, "worker", wid)
		return
	}
	tmp := output.TempPath(j.OutputPath)
	start := time.Now()
	res, err := o.transcoder.Transcode(ctx, o.codec, j.InputPath, tmp, o.quality)
	if err != nil {
		o.fail(ctx, id, err)
		return
	}
	if err := output.CommitTempFile(tmp, j.OutputPath); err != nil {
		o.fail(ctx, id, err)
		return
	}
	o.metrics.AddBytes(res.BytesIn, res.BytesOut)
	o.metrics.IncProcessed()
	_ = o.store.SetJobOutcome(ctx, id, jobs.StateSuccess, "", res.BytesIn, res.BytesOut, time.Since(start).Milliseconds(), nil)
	if h, err := o.hasher.HashFile(ctx, j.InputPath); err == nil {
		_ = o.dedupe.Put(ctx, h, j.OutputPath)
	}
}

func (o *Orchestrator) fail(ctx context.Context, id string, err error) {
	msg := err.Error()
	o.metrics.IncFailed()
	_ = o.store.SetJobOutcome(ctx, id, jobs.StateFailed, "", 0, 0, 0, &msg)
}

func extForCodec(c jobs.ImageCodec) string {
	if c == jobs.CodecAVIF {
		return ".avif"
	}
	return ".webp"
}
