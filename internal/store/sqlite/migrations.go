package sqlite

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY);`,
	`INSERT OR IGNORE INTO schema_migrations(version) VALUES(1);`,
	`CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		input_path TEXT NOT NULL,
		output_path TEXT NOT NULL,
		codec TEXT NOT NULL,
		quality INTEGER NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		state TEXT NOT NULL,
		skip_reason TEXT NOT NULL DEFAULT '',
		error_message TEXT NULL,
		bytes_in INTEGER NOT NULL DEFAULT 0,
		bytes_out INTEGER NOT NULL DEFAULT 0,
		duration_ms INTEGER NOT NULL DEFAULT 0
	);`,
	`CREATE INDEX IF NOT EXISTS idx_jobs_input_path ON jobs(input_path);`,
	`CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at);`,
	`CREATE TABLE IF NOT EXISTS file_hashes (
		input_path TEXT NOT NULL,
		codec TEXT NOT NULL,
		quality INTEGER NOT NULL,
		fingerprint TEXT NOT NULL,
		sha256 TEXT,
		output_path TEXT NOT NULL,
		created_at TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_file_hashes_fingerprint ON file_hashes(fingerprint);`,
	`CREATE INDEX IF NOT EXISTS idx_file_hashes_sha256 ON file_hashes(sha256);`,
	`CREATE INDEX IF NOT EXISTS idx_file_hashes_input_path ON file_hashes(input_path);`,
}
