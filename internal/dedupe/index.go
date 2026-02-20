package dedupe

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type DedupeIndex interface {
	Put(ctx context.Context, hash string, outputPath string) error
	Get(ctx context.Context, hash string) (outputPath string, ok bool, err error)
}

type SQLiteIndex struct{ DBPath string }

func NewSQLiteIndex(dbPath string) *SQLiteIndex { return &SQLiteIndex{DBPath: dbPath} }

func (s *SQLiteIndex) Put(ctx context.Context, hash string, outputPath string) error {
	q := fmt.Sprintf(`INSERT INTO file_hashes(input_path,codec,quality,fingerprint,sha256,output_path,created_at)
VALUES('','',0,'','%s','%s','%s');`, esc(hash), esc(outputPath), time.Now().UTC().Format(time.RFC3339Nano))
	_, err := s.exec(ctx, q)
	return err
}

func (s *SQLiteIndex) Get(ctx context.Context, hash string) (string, bool, error) {
	out, err := s.exec(ctx, fmt.Sprintf(`SELECT output_path FROM file_hashes WHERE sha256='%s' ORDER BY created_at DESC LIMIT 1;`, esc(hash)))
	if err != nil {
		return "", false, err
	}
	line := strings.TrimSpace(out)
	if line == "" {
		return "", false, nil
	}
	return line, true, nil
}

func (s *SQLiteIndex) exec(ctx context.Context, q string) (string, error) {
	cmd := exec.CommandContext(ctx, "sqlite3", "-cmd", ".timeout 2000", s.DBPath, q)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("sqlite: %w: %s", err, strings.TrimSpace(string(b)))
	}
	return string(b), nil
}

func esc(v string) string { return strings.ReplaceAll(v, "'", "''") }
