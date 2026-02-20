package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mediacrunch/internal/jobs"
	"mediacrunch/pkg/codec/image"
)

type Transcoder interface {
	Transcode(ctx context.Context, codec jobs.ImageCodec, inPath, outPath string, quality int) (image.Result, error)
}

type Report struct {
	Codec            jobs.ImageCodec `json:"codec"`
	Runs             int             `json:"runs"`
	FilesProcessed   int             `json:"files_processed"`
	TotalBytesIn     int64           `json:"bytes_in"`
	TotalBytesOut    int64           `json:"bytes_out"`
	TotalDurationMs  int64           `json:"total_duration_ms"`
	ThroughputPerSec float64         `json:"throughput_files_per_sec"`
	CompressionRatio float64         `json:"compression_ratio"`
}

func Run(ctx context.Context, codec jobs.ImageCodec, inputDir, outDir string, quality, runs int, tr Transcoder) (Report, error) {
	if runs <= 0 {
		runs = 1
	}
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return Report{}, err
	}
	files := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			files = append(files, filepath.Join(inputDir, e.Name()))
		}
	}
	sort.Strings(files)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return Report{}, err
	}
	rep := Report{Codec: codec, Runs: runs}
	startAll := time.Now()
	for i := 0; i < runs; i++ {
		runDir := filepath.Join(outDir, fmt.Sprintf("run-%02d", i+1))
		_ = os.RemoveAll(runDir)
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			return Report{}, err
		}
		for _, in := range files {
			out := filepath.Join(runDir, strings.TrimSuffix(filepath.Base(in), filepath.Ext(in))+extFor(codec))
			res, err := tr.Transcode(ctx, codec, in, out, quality)
			if err != nil {
				return Report{}, err
			}
			rep.FilesProcessed++
			rep.TotalBytesIn += res.BytesIn
			rep.TotalBytesOut += res.BytesOut
		}
	}
	rep.TotalDurationMs = time.Since(startAll).Milliseconds()
	if rep.TotalDurationMs > 0 {
		rep.ThroughputPerSec = float64(rep.FilesProcessed) / (float64(rep.TotalDurationMs) / 1000)
	}
	if rep.TotalBytesIn > 0 {
		rep.CompressionRatio = float64(rep.TotalBytesOut) / float64(rep.TotalBytesIn)
	}
	b, _ := json.MarshalIndent(rep, "", "  ")
	if err := os.WriteFile(filepath.Join(outDir, "report.json"), b, 0o644); err != nil {
		return Report{}, err
	}
	return rep, nil
}

func extFor(codec jobs.ImageCodec) string {
	if codec == jobs.CodecAVIF {
		return ".avif"
	}
	return ".webp"
}
