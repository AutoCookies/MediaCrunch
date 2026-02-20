package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"mediacrunch/internal/bench"
	"mediacrunch/internal/config"
	"mediacrunch/internal/daemon"
	"mediacrunch/internal/dedupe"
	"mediacrunch/internal/jobs"
	"mediacrunch/internal/logging"
	"mediacrunch/internal/metrics"
	"mediacrunch/internal/orchestrator"
	"mediacrunch/internal/skip"
	"mediacrunch/internal/store/sqlite"
	"mediacrunch/internal/watcher"
	"mediacrunch/pkg/codec/image"
)

type ImageTranscoder interface {
	Transcode(ctx context.Context, codec jobs.ImageCodec, inPath, outPath string, quality int) (image.Result, error)
}

type App struct{ transcoder ImageTranscoder }

func New(transcoder ImageTranscoder) *App { return &App{transcoder: transcoder} }

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("missing command")
	}
	switch args[0] {
	case "transcode":
		cfg, err := config.ParseTranscodeArgs(args[1:])
		if err != nil {
			return err
		}
		_, err = a.transcoder.Transcode(context.Background(), cfg.Codec, cfg.InPath, cfg.OutPath, cfg.Quality)
		return err
	case "daemon":
		cfg, err := config.ParseDaemonArgs(args[1:])
		if err != nil {
			return err
		}
		if cfg.PprofAddr != "" {
			go http.ListenAndServe(cfg.PprofAddr, nil)
		}
		logger := logging.New()
		m := metrics.New()
		st, _ := sqlite.New(cfg.DBPath)
		defer st.Close()
		if err := st.Init(context.Background()); err != nil {
			return err
		}
		w, err := watcher.New(cfg.InputDir)
		if err != nil {
			return err
		}
		q := jobs.NewQueue(cfg.QueueSize)
		hasher := dedupe.NewHasher(256 << 20)
		idx := dedupe.NewSQLiteIndex(cfg.DBPath)
		sp := skip.NewPolicy(st, idx, hasher, cfg.Codec, cfg.Quality)
		orch := orchestrator.New(q, st, a.transcoder, m, logger, cfg.Workers, cfg.Codec, cfg.Quality, cfg.OutputDir, sp, idx, hasher)
		d := daemon.New(w, orch, cfg.MetricsAddr, cfg.InputDir, logger)
		return d.Run(context.Background(), m.Handler())
	case "bench":
		cfg, err := config.ParseBenchArgs(args[1:])
		if err != nil {
			return err
		}
		if cfg.PprofAddr != "" {
			go http.ListenAndServe(cfg.PprofAddr, nil)
		}
		_, err = bench.Run(context.Background(), cfg.Codec, cfg.InputDir, cfg.OutDir, cfg.Quality, cfg.Runs, a.transcoder)
		return err
	default:
		return fmt.Errorf("unsupported command %q", args[0])
	}
}
