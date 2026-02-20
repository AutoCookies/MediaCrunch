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

	if err := cfg.Validate(); err != nil {
		return TranscodeConfig{}, err
	}

	return cfg, nil
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

	inputExt := strings.ToLower(filepath.Ext(c.InPath))
	switch inputExt {
	case ".jpg", ".jpeg", ".png":
	default:
		return fmt.Errorf("unsupported input extension %q: use .jpg, .jpeg, or .png", inputExt)
	}

	if strings.ToLower(filepath.Ext(c.OutPath)) != ".webp" {
		return fmt.Errorf("unsupported output extension %q: only .webp is supported", filepath.Ext(c.OutPath))
	}

	if _, err := os.Stat(c.InPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("input file does not exist: %s", c.InPath)
		}
		return fmt.Errorf("stat input file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(c.OutPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	return nil
}
