package store

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type Job struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Logs      string `json:"logs"`
	Error     string `json:"error"`
	CreatedAt string `json:"created_at"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at"`
}

type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS jobs (
id INTEGER PRIMARY KEY AUTOINCREMENT,
name TEXT NOT NULL,
status TEXT NOT NULL DEFAULT 'queued',
logs TEXT NOT NULL DEFAULT '',
error TEXT NOT NULL DEFAULT '',
created_at TEXT NOT NULL,
started_at TEXT NOT NULL DEFAULT '',
ended_at TEXT NOT NULL DEFAULT ''
)
`)

	if err != nil {
		db.Close()
		return nil, err
	}

	_, err = db.Exec(`
UPDATE jobs
SET status='failed', error='Runner restarted', ended_at=?
WHERE status IN ('queued', 'running')
`, now())

	if err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func (s *Store) Create(name string) (int64, error) {
	result, err := s.db.Exec(
		"INSERT INTO jobs (name, created_at) VALUES (?, ?)",
		name, now(),
	)

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (s *Store) Start(id int64) error {
	_, err := s.db.Exec(
		"UPDATE jobs SET status='running', started_at=? WHERE id=?",
		now(), id,
	)
	return err
}

func (s *Store) AppendLogs(id int64, logs string) error {
	_, err := s.db.Exec(`
UPDATE jobs
SET logs=substr(logs || ?, 1, 65536)
WHERE id=?
`, logs, id)

	return err
}

func (s *Store) Finish(id int64, status, message string) error {
	_, err := s.db.Exec(`
UPDATE jobs
SET status=?, error=?, ended_at=?
WHERE id=?
`, status, message, now(), id)

	return err
}

func (s *Store) Get(id int64) (Job, error) {
	var j Job

	err := s.db.QueryRow(`
SELECT id, name, status, logs, error,
       created_at, started_at, ended_at
FROM jobs WHERE id=?
`, id).Scan(
		&j.ID, &j.Name, &j.Status, &j.Logs,
		&j.Error, &j.CreatedAt, &j.StartedAt, &j.EndedAt,
	)

	return j, err
}

func (s *Store) List() ([]Job, error) {
	rows, err := s.db.Query(`
SELECT id, name, status, '', error,
       created_at, started_at, ended_at
FROM jobs ORDER BY id DESC LIMIT 100
`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []Job{}

	for rows.Next() {
		var j Job

		err := rows.Scan(
			&j.ID, &j.Name, &j.Status, &j.Logs,
			&j.Error, &j.CreatedAt, &j.StartedAt, &j.EndedAt,
		)

		if err != nil {
			return nil, err
		}

		jobs = append(jobs, j)
	}

	return jobs, rows.Err()
}

func (s *Store) Close() error {
	return s.db.Close()
}
