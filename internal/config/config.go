package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mediacrunch/internal/jobs"
)

type TranscodeConfig struct {
	InPath, OutPath string
	Quality         int
	Codec           jobs.ImageCodec
}
type DaemonConfig struct {
	InputDir, OutputDir, DBPath                  string
	Workers, QueueSize, Quality                  int
	Codec                                        jobs.ImageCodec
	WriteMode, OnSuccess, MetricsAddr, PprofAddr string
}
type BenchConfig struct {
	InputDir, OutDir string
	Runs, Quality    int
	Codec            jobs.ImageCodec
	PprofAddr        string
}

func ParseTranscodeArgs(args []string) (TranscodeConfig, error) {
	cfg := TranscodeConfig{Quality: 82, Codec: jobs.CodecWebP}
	fs := flag.NewFlagSet("transcode", flag.ContinueOnError)
	fs.StringVar(&cfg.InPath, "in", "", "")
	fs.StringVar(&cfg.OutPath, "out", "", "")
	fs.IntVar(&cfg.Quality, "q", 82, "")
	fs.StringVar((*string)(&cfg.Codec), "codec", string(cfg.Codec), "webp|avif")
	if err := fs.Parse(args); err != nil {
		return TranscodeConfig{}, err
	}
	return cfg, cfg.Validate()
}
func ParseDaemonArgs(args []string) (DaemonConfig, error) {
	cfg := DaemonConfig{Workers: 4, QueueSize: 256, Quality: 82, Codec: jobs.CodecWebP, WriteMode: "atomic", OnSuccess: "keep", MetricsAddr: "127.0.0.1:9090"}
	fs := flag.NewFlagSet("daemon", flag.ContinueOnError)
	fs.StringVar(&cfg.InputDir, "input", "", "")
	fs.StringVar(&cfg.OutputDir, "output", "", "")
	fs.StringVar(&cfg.DBPath, "db", "", "")
	fs.IntVar(&cfg.Workers, "workers", cfg.Workers, "")
	fs.IntVar(&cfg.QueueSize, "queue-size", cfg.QueueSize, "")
	fs.IntVar(&cfg.Quality, "quality", cfg.Quality, "")
	fs.StringVar((*string)(&cfg.Codec), "codec", string(cfg.Codec), "webp|avif")
	fs.StringVar(&cfg.WriteMode, "write-mode", cfg.WriteMode, "")
	fs.StringVar(&cfg.OnSuccess, "on-success", cfg.OnSuccess, "")
	fs.StringVar(&cfg.MetricsAddr, "metrics-addr", cfg.MetricsAddr, "")
	fs.StringVar(&cfg.PprofAddr, "pprof-addr", "", "")
	if err := fs.Parse(args); err != nil {
		return DaemonConfig{}, err
	}
	return cfg, cfg.Validate()
}
func ParseBenchArgs(args []string) (BenchConfig, error) {
	cfg := BenchConfig{Runs: 3, Quality: 82, Codec: jobs.CodecWebP}
	fs := flag.NewFlagSet("bench", flag.ContinueOnError)
	fs.StringVar(&cfg.InputDir, "input", "", "")
	fs.StringVar(&cfg.OutDir, "out", "", "")
	fs.IntVar(&cfg.Runs, "runs", cfg.Runs, "")
	fs.IntVar(&cfg.Quality, "q", cfg.Quality, "")
	fs.StringVar((*string)(&cfg.Codec), "codec", string(cfg.Codec), "webp|avif")
	fs.StringVar(&cfg.PprofAddr, "pprof-addr", "", "")
	if err := fs.Parse(args); err != nil {
		return BenchConfig{}, err
	}
	return cfg, cfg.Validate()
}

func (c TranscodeConfig) Validate() error {
	if c.InPath == "" || c.OutPath == "" {
		return errors.New("--in and --out are required")
	}
	if c.Quality < 1 || c.Quality > 100 {
		return fmt.Errorf("invalid quality %d", c.Quality)
	}
	if c.Codec != jobs.CodecWebP && c.Codec != jobs.CodecAVIF {
		return errors.New("unsupported codec")
	}
	if err := validateImageInput(c.InPath); err != nil {
		return err
	}
	ext := strings.ToLower(filepath.Ext(c.OutPath))
	if (c.Codec == jobs.CodecWebP && ext != ".webp") || (c.Codec == jobs.CodecAVIF && ext != ".avif") {
		return errors.New("output extension does not match codec")
	}
	return os.MkdirAll(filepath.Dir(c.OutPath), 0o755)
}
func (c DaemonConfig) Validate() error {
	if c.InputDir == "" || c.OutputDir == "" || c.DBPath == "" {
		return errors.New("--input,--output,--db required")
	}
	if c.Workers <= 0 || c.QueueSize <= 0 {
		return errors.New("invalid workers/queue-size")
	}
	if c.Quality < 1 || c.Quality > 100 {
		return errors.New("invalid quality")
	}
	if c.Codec != jobs.CodecWebP && c.Codec != jobs.CodecAVIF {
		return errors.New("unsupported codec")
	}
	if c.WriteMode != "atomic" || c.OnSuccess != "keep" {
		return errors.New("unsupported write-mode/on-success")
	}
	_ = os.MkdirAll(c.InputDir, 0o755)
	_ = os.MkdirAll(c.OutputDir, 0o755)
	_ = os.MkdirAll(filepath.Dir(c.DBPath), 0o755)
	return nil
}
func (c BenchConfig) Validate() error {
	if c.InputDir == "" || c.OutDir == "" {
		return errors.New("--input and --out required")
	}
	if c.Runs <= 0 {
		return errors.New("runs must be >0")
	}
	if c.Codec != jobs.CodecWebP && c.Codec != jobs.CodecAVIF {
		return errors.New("unsupported codec")
	}
	_ = os.MkdirAll(c.OutDir, 0o755)
	return nil
}

func validateImageInput(in string) error {
	ext := strings.ToLower(filepath.Ext(in))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return errors.New("unsupported input extension")
	}
	_, err := os.Stat(in)
	return err
}
