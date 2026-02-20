package output

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
)

func TempPath(finalPath string) string {
	return fmt.Sprintf("%s.tmp.%d.%d", finalPath, os.Getpid(), rand.Int63())
}

func CommitTempFile(tempPath, finalPath string) error {
	f, err := os.OpenFile(tempPath, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open temp file: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("fsync temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return fmt.Errorf("rename temp to final: %w", err)
	}
	df, err := os.Open(filepath.Dir(finalPath))
	if err == nil {
		_ = df.Sync()
		_ = df.Close()
	}
	return nil
}
