package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type TranscodeConfig struct {
	InPath  string
	OutPath string
	Quality int
}

type DaemonConfig struct {
	InputDir    string
	OutputDir   string
	DBPath      string
	Workers     int
	QueueSize   int
	Quality     int
	WriteMode   string
	OnSuccess   string
	MetricsAddr string
}

func ParseTranscodeArgs(args []string) (TranscodeConfig, error) {
	fs := flag.NewFlagSet("transcode", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	cfg := TranscodeConfig{}
	fs.StringVar(&cfg.InPath, "in", "", "input image path (.jpg, .jpeg, .png)")
	fs.StringVar(&cfg.OutPath, "out", "", "output image path (.webp)")
	fs.IntVar(&cfg.Quality, "q", 82, "WebP quality [1-100]")

	if err := fs.Parse(args); err != nil {
		return TranscodeConfig{}, fmt.Errorf("parse transcode flags: %w", err)
	}
	if fs.NArg() != 0 {
		return TranscodeConfig{}, fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	return cfg, cfg.Validate()
}

func ParseDaemonArgs(args []string) (DaemonConfig, error) {
	cfg := DaemonConfig{Workers: 4, QueueSize: 256, Quality: 82, WriteMode: "atomic", OnSuccess: "keep", MetricsAddr: "127.0.0.1:9090"}
	fs := flag.NewFlagSet("daemon", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&cfg.InputDir, "input", "", "input folder")
	fs.StringVar(&cfg.OutputDir, "output", "", "output folder")
	fs.StringVar(&cfg.DBPath, "db", "", "sqlite db path")
	fs.IntVar(&cfg.Workers, "workers", cfg.Workers, "worker count")
	fs.IntVar(&cfg.QueueSize, "queue-size", cfg.QueueSize, "bounded queue size")
	fs.IntVar(&cfg.Quality, "quality", cfg.Quality, "quality 1-100")
	fs.StringVar(&cfg.WriteMode, "write-mode", cfg.WriteMode, "atomic")
	fs.StringVar(&cfg.OnSuccess, "on-success", cfg.OnSuccess, "keep")
	fs.StringVar(&cfg.MetricsAddr, "metrics-addr", cfg.MetricsAddr, "metrics listen address")
	if err := fs.Parse(args); err != nil {
		return DaemonConfig{}, fmt.Errorf("parse daemon flags: %w", err)
	}
	if fs.NArg() != 0 {
		return DaemonConfig{}, fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	return cfg, cfg.Validate()
}

func (c TranscodeConfig) Validate() error {
	if c.InPath == "" {
		return errors.New("--in is required")
	}
	if c.OutPath == "" {
		return errors.New("--out is required")
	}
	if c.Quality < 1 || c.Quality > 100 {
		return fmt.Errorf("invalid quality %d: must be between 1 and 100", c.Quality)
	}
	if err := validateImageInput(c.InPath); err != nil {
		return err
	}
	if strings.ToLower(filepath.Ext(c.OutPath)) != ".webp" {
		return fmt.Errorf("unsupported output extension %q: only .webp is supported", filepath.Ext(c.OutPath))
	}
	if err := os.MkdirAll(filepath.Dir(c.OutPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	return nil
}

func (c DaemonConfig) Validate() error {
	if c.InputDir == "" || c.OutputDir == "" || c.DBPath == "" {
		return errors.New("--input, --output, and --db are required")
	}
	if c.Workers <= 0 {
		return errors.New("--workers must be > 0")
	}
	if c.QueueSize <= 0 {
		return errors.New("--queue-size must be > 0")
	}
	if c.Quality < 1 || c.Quality > 100 {
		return fmt.Errorf("invalid quality %d: must be between 1 and 100", c.Quality)
	}
	if c.WriteMode != "atomic" {
		return errors.New("--write-mode must be atomic")
	}
	if c.OnSuccess != "keep" {
		return errors.New("--on-success must be keep")
	}
	if err := os.MkdirAll(c.InputDir, 0o755); err != nil {
		return fmt.Errorf("create input dir: %w", err)
	}
	if err := os.MkdirAll(c.OutputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(c.DBPath), 0o755); err != nil {
		return fmt.Errorf("create db dir: %w", err)
	}
	return nil
}

func validateImageInput(inPath string) error {
	inputExt := strings.ToLower(filepath.Ext(inPath))
	switch inputExt {
	case ".jpg", ".jpeg", ".png":
	default:
		return fmt.Errorf("unsupported input extension %q: use .jpg, .jpeg, or .png", inputExt)
	}
	if _, err := os.Stat(inPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("input file does not exist: %s", inPath)
		}
		return fmt.Errorf("stat input file: %w", err)
	}
	return nil
}
