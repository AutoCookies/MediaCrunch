package app

import (
	"context"
	"errors"
	"fmt"

	"mediacrunch/internal/config"
	"mediacrunch/internal/daemon"
	"mediacrunch/internal/jobs"
	"mediacrunch/internal/logging"
	"mediacrunch/internal/metrics"
	"mediacrunch/internal/orchestrator"
	"mediacrunch/internal/store/sqlite"
	"mediacrunch/internal/watcher"
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
		return errors.New("missing command: supported commands are 'transcode' and 'daemon'")
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
	case "daemon":
		cfg, err := config.ParseDaemonArgs(args[1:])
		if err != nil {
			return err
		}
		logger := logging.New()
		m := metrics.New()
		st, err := sqlite.New(cfg.DBPath)
		if err != nil {
			return err
		}
		defer st.Close()
		if err := st.Init(context.Background()); err != nil {
			return err
		}
		w, err := watcher.New(cfg.InputDir)
		if err != nil {
			return err
		}
		q := jobs.NewQueue(cfg.QueueSize)
		orch := orchestrator.New(q, st, a.transcoder, m, logger, cfg.Workers, cfg.Quality, cfg.OutputDir)
		d := daemon.New(w, orch, cfg.MetricsAddr, cfg.InputDir, logger)
		return d.Run(context.Background(), m.Handler())
	default:
		return fmt.Errorf("unsupported command %q", args[0])
	}
}
