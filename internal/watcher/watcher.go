package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Watcher interface {
	Start(ctx context.Context) (<-chan string, <-chan error)
	Close() error
}

type FSWatcher struct {
	inputDir string
	stopCh   chan struct{}
}

func New(inputDir string) (*FSWatcher, error) {
	if st, err := os.Stat(inputDir); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("input dir invalid: %s", inputDir)
	}
	return &FSWatcher{inputDir: inputDir, stopCh: make(chan struct{})}, nil
}

func (f *FSWatcher) Start(ctx context.Context) (<-chan string, <-chan error) {
	paths := make(chan string, 16)
	errs := make(chan error, 1)
	known := map[string]time.Time{}
	go func() {
		defer close(paths)
		defer close(errs)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-f.stopCh:
				return
			case <-ticker.C:
				entries, err := os.ReadDir(f.inputDir)
				if err != nil {
					errs <- err
					continue
				}
				for _, e := range entries {
					if e.IsDir() {
						continue
					}
					name := e.Name()
					ext := strings.ToLower(filepath.Ext(name))
					if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
						continue
					}
					full := filepath.Join(f.inputDir, name)
					st, err := os.Stat(full)
					if err != nil {
						continue
					}
					if prev, ok := known[full]; !ok || st.ModTime().After(prev) {
						known[full] = st.ModTime()
						paths <- full
					}
				}
			}
		}
	}()
	return paths, errs
}

func (f *FSWatcher) Close() error {
	close(f.stopCh)
	return nil
}
