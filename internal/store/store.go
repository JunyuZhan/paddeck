package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Sample is one minute-aggregated host metric row.
type Sample struct {
	Ts        int64   `json:"ts"`
	CPU       float64 `json:"cpu"`
	MemUsed   uint64  `json:"mem_used"`
	MemTotal  uint64  `json:"mem_total"`
	DiskUsed  uint64  `json:"disk_used"`
	DiskTotal uint64  `json:"disk_total"`
	Load1     float64 `json:"load1"`
}

// Store wraps the SQLite history database.
type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS host_samples (
  host_id    TEXT    NOT NULL,
  ts         INTEGER NOT NULL,
  cpu        REAL    NOT NULL DEFAULT 0,
  mem_used   INTEGER NOT NULL DEFAULT 0,
  mem_total  INTEGER NOT NULL DEFAULT 0,
  disk_used  INTEGER NOT NULL DEFAULT 0,
  disk_total INTEGER NOT NULL DEFAULT 0,
  load1      REAL    NOT NULL DEFAULT 0,
  PRIMARY KEY (host_id, ts)
);
CREATE INDEX IF NOT EXISTS idx_host_ts ON host_samples(host_id, ts);
`)
	return err
}

// Insert writes a sample aligned to the minute. Duplicate minutes are replaced.
func (s *Store) Insert(hostID string, ts int64, cpu float64, memUsed, memTotal, diskUsed, diskTotal uint64, load1 float64) error {
	bucket := ts - ts%60
	_, err := s.db.Exec(`
INSERT INTO host_samples (host_id, ts, cpu, mem_used, mem_total, disk_used, disk_total, load1)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(host_id, ts) DO UPDATE SET
  cpu=excluded.cpu, mem_used=excluded.mem_used, mem_total=excluded.mem_total,
  disk_used=excluded.disk_used, disk_total=excluded.disk_total, load1=excluded.load1`,
		hostID, bucket, cpu, memUsed, memTotal, diskUsed, diskTotal, load1)
	return err
}

// History returns samples in [from, to], downsampled to at most maxPoints buckets.
func (s *Store) History(hostID string, from, to int64, maxPoints int) ([]Sample, error) {
	if maxPoints <= 0 {
		maxPoints = 240
	}
	span := to - from
	if span <= 0 {
		return nil, nil
	}
	step := span / int64(maxPoints)
	if step < 60 {
		step = 60
	}
	rows, err := s.db.Query(`
SELECT (ts/?)*? AS bucket,
       AVG(cpu), AVG(mem_used), AVG(mem_total), AVG(disk_used), AVG(disk_total), AVG(load1)
FROM host_samples
WHERE host_id = ? AND ts >= ? AND ts <= ?
GROUP BY bucket ORDER BY bucket`, step, step, hostID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Sample
	for rows.Next() {
		var v Sample
		var cpu, memUsed, memTotal, diskUsed, diskTotal, load1 float64
		if err := rows.Scan(&v.Ts, &cpu, &memUsed, &memTotal, &diskUsed, &diskTotal, &load1); err != nil {
			return nil, err
		}
		v.CPU = cpu
		v.MemUsed = uint64(memUsed)
		v.MemTotal = uint64(memTotal)
		v.DiskUsed = uint64(diskUsed)
		v.DiskTotal = uint64(diskTotal)
		v.Load1 = load1
		out = append(out, v)
	}
	return out, rows.Err()
}

// Prune deletes samples older than the retention window.
func (s *Store) Prune(retentionDays int) error {
	if retentionDays <= 0 {
		retentionDays = 30
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays).Unix()
	_, err := s.db.Exec(`DELETE FROM host_samples WHERE ts < ?`, cutoff)
	return err
}

func (s *Store) Close() error { return s.db.Close() }
