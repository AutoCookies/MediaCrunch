package skip

import (
	"context"
	"os"
	"strings"

	"mediacrunch/internal/dedupe"
	"mediacrunch/internal/jobs"
	"mediacrunch/internal/store"
)

type SkipPolicy interface {
	ShouldSkip(ctx context.Context, inputPath, outputPath string) (skip bool, reason string, err error)
}

type Policy struct {
	Store   store.JobStore
	Dedupe  dedupe.DedupeIndex
	Hasher  dedupe.Hasher
	Codec   jobs.ImageCodec
	Quality int
}

func NewPolicy(st store.JobStore, idx dedupe.DedupeIndex, hasher dedupe.Hasher, codec jobs.ImageCodec, quality int) *Policy {
	return &Policy{Store: st, Dedupe: idx, Hasher: hasher, Codec: codec, Quality: quality}
}

func (p *Policy) ShouldSkip(ctx context.Context, inputPath, outputPath string) (bool, string, error) {
	in, err := os.Stat(inputPath)
	if err != nil {
		return false, "", err
	}
	if out, err := os.Stat(outputPath); err == nil && !out.ModTime().Before(in.ModTime()) && validByCodec(p.Codec, outputPath) {
		return true, "output_fresh", nil
	}
	latest, err := p.Store.FindLatestByInputPath(ctx, inputPath)
	if err != nil {
		return false, "", err
	}
	if latest != nil && latest.State == jobs.StateSuccess && latest.Codec == p.Codec && latest.Quality == p.Quality && validByCodec(p.Codec, latest.OutputPath) {
		return true, "store_success", nil
	}
	hash, err := p.Hasher.HashFile(ctx, inputPath)
	if err != nil {
		return false, "", err
	}
	if existing, ok, err := p.Dedupe.Get(ctx, hash); err != nil {
		return false, "", err
	} else if ok && validByCodec(p.Codec, existing) {
		return true, "deduped_hash", nil
	}
	return false, "", nil
}

func validByCodec(codec jobs.ImageCodec, path string) bool {
	b, err := os.ReadFile(path)
	if err != nil || len(b) < 32 {
		return false
	}
	if codec == jobs.CodecWebP {
		return string(b[:4]) == "RIFF" && strings.Contains(string(b[:32]), "WEBP")
	}
	if len(b) < 16 || string(b[4:8]) != "ftyp" {
		return false
	}
	head := string(b[:32])
	return strings.Contains(head, "avif") || strings.Contains(head, "avis")
}
