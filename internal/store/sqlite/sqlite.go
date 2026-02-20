package sqlite

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"mediacrunch/internal/jobs"
)

type Store struct {
	path string
}

func New(path string) (*Store, error) { return &Store{path: path}, nil }

func (s *Store) Init(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY);`,
		`INSERT OR IGNORE INTO schema_migrations(version) VALUES(1);`,
		`CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			input_path TEXT NOT NULL,
			output_path TEXT NOT NULL,
			quality INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			state TEXT NOT NULL,
			error_message TEXT NULL,
			bytes_in INTEGER NOT NULL DEFAULT 0,
			bytes_out INTEGER NOT NULL DEFAULT 0,
			duration_ms INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_input_path ON jobs(input_path);`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at);`,
	}
	for _, stmt := range stmts {
		if err := s.exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CreateJob(ctx context.Context, j jobs.Job) (jobs.Job, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	j.CreatedAt = time.Now().UTC()
	j.UpdatedAt = j.CreatedAt
	errMsg := "NULL"
	if j.ErrorMessage != nil {
		errMsg = "'" + esc(*j.ErrorMessage) + "'"
	}
	q := fmt.Sprintf(`INSERT INTO jobs(id,input_path,output_path,quality,created_at,updated_at,state,error_message,bytes_in,bytes_out,duration_ms)
VALUES('%s','%s','%s',%d,'%s','%s','%s',%s,%d,%d,%d);`, esc(j.ID), esc(j.InputPath), esc(j.OutputPath), j.Quality, now, now, esc(string(j.State)), errMsg, j.BytesIn, j.BytesOut, j.DurationMs)
	if err := s.exec(ctx, q); err != nil {
		return jobs.Job{}, err
	}
	return j, nil
}

func (s *Store) UpdateJobState(ctx context.Context, id string, state jobs.JobState, errMsg *string) error {
	cur, err := s.GetJob(ctx, id)
	if err != nil {
		return err
	}
	if err := jobs.ValidateTransition(cur.State, state); err != nil {
		return err
	}
	errSQL := "NULL"
	if errMsg != nil {
		errSQL = "'" + esc(*errMsg) + "'"
	}
	q := fmt.Sprintf(`UPDATE jobs SET state='%s', error_message=%s, updated_at='%s' WHERE id='%s';`, esc(string(state)), errSQL, time.Now().UTC().Format(time.RFC3339Nano), esc(id))
	return s.exec(ctx, q)
}

func (s *Store) GetJob(ctx context.Context, id string) (jobs.Job, error) {
	rows, err := s.query(ctx, fmt.Sprintf(`SELECT id,input_path,output_path,quality,created_at,updated_at,state,COALESCE(error_message,''),bytes_in,bytes_out,duration_ms FROM jobs WHERE id='%s' LIMIT 1;`, esc(id)))
	if err != nil {
		return jobs.Job{}, err
	}
	if len(rows) == 0 {
		return jobs.Job{}, errors.New("job not found")
	}
	return parseJob(rows[0])
}

func (s *Store) FindLatestByInputPath(ctx context.Context, inPath string) (*jobs.Job, error) {
	rows, err := s.query(ctx, fmt.Sprintf(`SELECT id,input_path,output_path,quality,created_at,updated_at,state,COALESCE(error_message,''),bytes_in,bytes_out,duration_ms FROM jobs WHERE input_path='%s' ORDER BY created_at DESC LIMIT 1;`, esc(inPath)))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	j, err := parseJob(rows[0])
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (s *Store) Stats(ctx context.Context) (jobs.Stats, error) {
	rows, err := s.query(ctx, `SELECT COUNT(*),SUM(CASE WHEN state='success' THEN 1 ELSE 0 END),SUM(CASE WHEN state='failed' THEN 1 ELSE 0 END),SUM(CASE WHEN state='skipped' THEN 1 ELSE 0 END),COALESCE(SUM(bytes_in),0),COALESCE(SUM(bytes_out),0) FROM jobs;`)
	if err != nil {
		return jobs.Stats{}, err
	}
	if len(rows) == 0 {
		return jobs.Stats{}, nil
	}
	parts := strings.Split(rows[0], "\t")
	if len(parts) != 6 {
		return jobs.Stats{}, fmt.Errorf("unexpected stats row: %q", rows[0])
	}
	vals := make([]int64, 6)
	for i := range 6 {
		v, err := strconv.ParseInt(parts[i], 10, 64)
		if err != nil {
			return jobs.Stats{}, err
		}
		vals[i] = v
	}
	return jobs.Stats{TotalJobs: vals[0], SuccessJobs: vals[1], FailedJobs: vals[2], SkippedJobs: vals[3], TotalBytesIn: vals[4], TotalBytesOut: vals[5]}, nil
}

func (s *Store) Close() error { return nil }

func (s *Store) exec(ctx context.Context, sql string) error {
	cmd := exec.CommandContext(ctx, "sqlite3", s.path, sql)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sqlite exec failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *Store) query(ctx context.Context, sql string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "sqlite3", "-separator", "\t", s.path, sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sqlite query failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

func parseJob(row string) (jobs.Job, error) {
	parts := strings.Split(row, "\t")
	if len(parts) != 11 {
		return jobs.Job{}, fmt.Errorf("unexpected job row: %q", row)
	}
	quality, _ := strconv.Atoi(parts[3])
	bytesIn, _ := strconv.ParseInt(parts[8], 10, 64)
	bytesOut, _ := strconv.ParseInt(parts[9], 10, 64)
	duration, _ := strconv.ParseInt(parts[10], 10, 64)
	createdAt, _ := time.Parse(time.RFC3339Nano, parts[4])
	updatedAt, _ := time.Parse(time.RFC3339Nano, parts[5])
	j := jobs.Job{ID: parts[0], InputPath: parts[1], OutputPath: parts[2], Quality: quality, CreatedAt: createdAt, UpdatedAt: updatedAt, State: jobs.JobState(parts[6]), BytesIn: bytesIn, BytesOut: bytesOut, DurationMs: duration}
	if parts[7] != "" {
		msg := parts[7]
		j.ErrorMessage = &msg
	}
	return j, nil
}

func esc(s string) string { return strings.ReplaceAll(s, "'", "''") }
