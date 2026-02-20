package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mediacrunch/internal/orchestrator"
	"mediacrunch/internal/watcher"
)

type Daemon struct {
	watcher      watcher.Watcher
	orchestrator *orchestrator.Orchestrator
	metricsAddr  string
	logger       *slog.Logger
	inputDir     string
}

func New(w watcher.Watcher, o *orchestrator.Orchestrator, metricsAddr, inputDir string, logger *slog.Logger) *Daemon {
	return &Daemon{watcher: w, orchestrator: o, metricsAddr: metricsAddr, logger: logger, inputDir: inputDir}
}

func (d *Daemon) Run(ctx context.Context, metricsHandler http.Handler) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := http.NewServeMux()
	mux.Handle("/metrics", metricsHandler)
	httpServer := &http.Server{Addr: d.metricsAddr, Handler: mux}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			d.logger.Error("metrics server failed", "error", err)
			cancel()
		}
	}()
	d.logger.Info("daemon started", "metrics_addr", d.metricsAddr)

	paths, errs := d.watcher.Start(ctx)
	go d.orchestrator.Start(ctx)

	if err := d.enqueueExisting(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			shutdownCtx, c := context.WithTimeout(context.Background(), 2*time.Second)
			defer c()
			_ = httpServer.Shutdown(shutdownCtx)
			_ = d.watcher.Close()
			d.logger.Info("daemon stopped")
			return nil
		case err, ok := <-errs:
			if ok {
				d.logger.Error("watcher error", "error", err)
			}
		case path, ok := <-paths:
			if !ok {
				return nil
			}
			if err := d.orchestrator.Submit(ctx, path); err != nil {
				d.logger.Error("submit failed", "path", path, "error", err)
			}
		}
	}
}

func (d *Daemon) enqueueExisting(ctx context.Context) error {
	entries, err := os.ReadDir(d.inputDir)
	if err != nil {
		return fmt.Errorf("read input dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			continue
		}
		if err := d.orchestrator.Submit(ctx, filepath.Join(d.inputDir, e.Name())); err != nil {
			d.logger.Error("submit existing file failed", "file", e.Name(), "error", err)
		}
	}
	return nil
}
