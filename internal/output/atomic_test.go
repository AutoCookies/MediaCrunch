package output

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommitTempFile(t *testing.T) {
	dir := t.TempDir()
	final := filepath.Join(dir, "x.webp")
	tmp := TempPath(final)
	if err := os.WriteFile(tmp, []byte("RIFFxxxxWEBP"), 0o644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if err := CommitTempFile(tmp, final); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := os.Stat(final); err != nil {
		t.Fatalf("final missing: %v", err)
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatalf("temp should not exist")
	}
}
