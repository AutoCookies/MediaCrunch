package app

import (
	"context"
	"errors"
	"fmt"

	"mediacrunch/internal/config"
)

type ImageTranscoder interface {
	TranscodeToWebP(ctx context.Context, inPath, outPath string, quality int) error
}

type App struct {
	transcoder ImageTranscoder
}

func New(transcoder ImageTranscoder) *App {
	return &App{transcoder: transcoder}
}

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("missing command: supported command is 'transcode'")
	}

	switch args[0] {
	case "transcode":
		cfg, err := config.ParseTranscodeArgs(args[1:])
		if err != nil {
			return err
		}
		if err := a.transcoder.TranscodeToWebP(context.Background(), cfg.InPath, cfg.OutPath, cfg.Quality); err != nil {
			return fmt.Errorf("transcode failed: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported command %q", args[0])
	}
}
