package sqlite

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"mediacrunch/internal/jobs"
)

type Store struct {
	path string
	mu   sync.Mutex
}

func New(path string) (*Store, error) { return &Store{path: path}, nil }

func (s *Store) Init(ctx context.Context) error {
	for _, m := range migrations {
		if _, err := s.exec(ctx, m); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CreateJob(ctx context.Context, j jobs.Job) (jobs.Job, error) {
	now := time.Now().UTC()
	j.CreatedAt, j.UpdatedAt = now, now
	errMsg := "NULL"
	if j.ErrorMessage != nil {
		errMsg = "'" + esc(*j.ErrorMessage) + "'"
	}
	q := fmt.Sprintf(`INSERT INTO jobs(id,input_path,output_path,codec,quality,created_at,updated_at,state,skip_reason,error_message,bytes_in,bytes_out,duration_ms)
VALUES('%s','%s','%s','%s',%d,'%s','%s','%s','%s',%s,%d,%d,%d);`, esc(j.ID), esc(j.InputPath), esc(j.OutputPath), esc(string(j.Codec)), j.Quality, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), esc(string(j.State)), esc(j.SkipReason), errMsg, j.BytesIn, j.BytesOut, j.DurationMs)
	if _, err := s.exec(ctx, q); err != nil {
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
	_, err = s.exec(ctx, fmt.Sprintf(`UPDATE jobs SET state='%s', error_message=%s, updated_at='%s' WHERE id='%s';`, esc(string(state)), errSQL, time.Now().UTC().Format(time.RFC3339Nano), esc(id)))
	return err
}

func (s *Store) SetJobOutcome(ctx context.Context, id string, state jobs.JobState, skipReason string, bytesIn, bytesOut, durationMs int64, errMsg *string) error {
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
	_, err = s.exec(ctx, fmt.Sprintf(`UPDATE jobs SET state='%s', skip_reason='%s', bytes_in=%d, bytes_out=%d, duration_ms=%d, error_message=%s, updated_at='%s' WHERE id='%s';`, esc(string(state)), esc(skipReason), bytesIn, bytesOut, durationMs, errSQL, time.Now().UTC().Format(time.RFC3339Nano), esc(id)))
	return err
}

func (s *Store) GetJob(ctx context.Context, id string) (jobs.Job, error) {
	rows, err := s.query(ctx, fmt.Sprintf(`SELECT id,input_path,output_path,codec,quality,created_at,updated_at,state,skip_reason,COALESCE(error_message,''),bytes_in,bytes_out,duration_ms FROM jobs WHERE id='%s' LIMIT 1;`, esc(id)))
	if err != nil {
		return jobs.Job{}, err
	}
	if len(rows) == 0 {
		return jobs.Job{}, errors.New("job not found")
	}
	return parseJob(rows[0])
}

func (s *Store) FindLatestByInputPath(ctx context.Context, inPath string) (*jobs.Job, error) {
	rows, err := s.query(ctx, fmt.Sprintf(`SELECT id,input_path,output_path,codec,quality,created_at,updated_at,state,skip_reason,COALESCE(error_message,''),bytes_in,bytes_out,duration_ms FROM jobs WHERE input_path='%s' ORDER BY created_at DESC LIMIT 1;`, esc(inPath)))
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
	if err != nil || len(rows) == 0 {
		return jobs.Stats{}, err
	}
	p := strings.Split(rows[0], "\t")
	vals := make([]int64, 6)
	for i := 0; i < 6; i++ {
		vals[i], _ = strconv.ParseInt(p[i], 10, 64)
	}
	return jobs.Stats{TotalJobs: vals[0], SuccessJobs: vals[1], FailedJobs: vals[2], SkippedJobs: vals[3], TotalBytesIn: vals[4], TotalBytesOut: vals[5]}, nil
}
func (s *Store) Close() error { return nil }

func (s *Store) exec(ctx context.Context, q string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cmd := exec.CommandContext(ctx, "sqlite3", "-cmd", ".timeout 2000", s.path, q)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("sqlite exec failed: %w: %s", err, strings.TrimSpace(string(b)))
	}
	return string(b), nil
}
func (s *Store) query(ctx context.Context, q string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cmd := exec.CommandContext(ctx, "sqlite3", "-cmd", ".timeout 2000", "-separator", "\t", s.path, q)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sqlite query failed: %w: %s", err, strings.TrimSpace(string(b)))
	}
	text := strings.TrimSpace(string(b))
	if text == "" {
		return nil, nil
	}
	return strings.Split(text, "\n"), nil
}

func parseJob(row string) (jobs.Job, error) {
	p := strings.Split(row, "\t")
	if len(p) != 13 {
		return jobs.Job{}, fmt.Errorf("unexpected row")
	}
	q, _ := strconv.Atoi(p[4])
	bi, _ := strconv.ParseInt(p[10], 10, 64)
	bo, _ := strconv.ParseInt(p[11], 10, 64)
	d, _ := strconv.ParseInt(p[12], 10, 64)
	ct, _ := time.Parse(time.RFC3339Nano, p[5])
	ut, _ := time.Parse(time.RFC3339Nano, p[6])
	j := jobs.Job{ID: p[0], InputPath: p[1], OutputPath: p[2], Codec: jobs.ImageCodec(p[3]), Quality: q, CreatedAt: ct, UpdatedAt: ut, State: jobs.JobState(p[7]), SkipReason: p[8], BytesIn: bi, BytesOut: bo, DurationMs: d}
	if p[9] != "" {
		msg := p[9]
		j.ErrorMessage = &msg
	}
	return j, nil
}
func esc(s string) string { return strings.ReplaceAll(s, "'", "''") }
