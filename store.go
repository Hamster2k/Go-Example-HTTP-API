package main

import (
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("object not found")

type Object struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Store struct {
	db *sql.DB
}

func OpenStore(path string) (*Store, error) {
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// A single connection avoids write contention; plenty for a small API.
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS objects (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT NOT NULL,
		message    TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Ping() error { return s.db.Ping() }

func (s *Store) Create(name, message string) (Object, error) {
	now := time.Now().UTC().Truncate(time.Second)
	res, err := s.db.Exec(
		`INSERT INTO objects (name, message, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		name, message, formatTime(now), formatTime(now),
	)
	if err != nil {
		return Object{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Object{}, err
	}
	return Object{ID: id, Name: name, Message: message, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *Store) List(limit, offset int) ([]Object, error) {
	rows, err := s.db.Query(
		`SELECT id, name, message, created_at, updated_at FROM objects ORDER BY id LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	objects := []Object{}
	for rows.Next() {
		o, err := scanObject(rows)
		if err != nil {
			return nil, err
		}
		objects = append(objects, o)
	}
	return objects, rows.Err()
}

func (s *Store) Get(id int64) (Object, error) {
	row := s.db.QueryRow(
		`SELECT id, name, message, created_at, updated_at FROM objects WHERE id = ?`, id,
	)
	o, err := scanObject(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Object{}, ErrNotFound
	}
	return o, err
}

func (s *Store) Update(id int64, name, message string) (Object, error) {
	now := formatTime(time.Now().UTC().Truncate(time.Second))
	res, err := s.db.Exec(
		`UPDATE objects SET name = ?, message = ?, updated_at = ? WHERE id = ?`,
		name, message, now, id,
	)
	if err != nil {
		return Object{}, err
	}
	if n, err := res.RowsAffected(); err != nil {
		return Object{}, err
	} else if n == 0 {
		return Object{}, ErrNotFound
	}
	return s.Get(id)
}

func (s *Store) Delete(id int64) error {
	res, err := s.db.Exec(`DELETE FROM objects WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanObject(sc scanner) (Object, error) {
	var o Object
	var created, updated string
	if err := sc.Scan(&o.ID, &o.Name, &o.Message, &created, &updated); err != nil {
		return Object{}, err
	}
	var err error
	if o.CreatedAt, err = time.Parse(time.RFC3339, created); err != nil {
		return Object{}, err
	}
	if o.UpdatedAt, err = time.Parse(time.RFC3339, updated); err != nil {
		return Object{}, err
	}
	return o, nil
}

func formatTime(t time.Time) string { return t.Format(time.RFC3339) }
