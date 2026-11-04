package database

import "database/sql"

// Migrate creates the catalog tables when they are missing. It is safe to
// call on every startup, including an existing data.db that already has them.
func Migrate(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS categories (
	id string,
	name string,
	description string
);
CREATE TABLE IF NOT EXISTS courses (
	id string,
	name string,
	description string,
	category_id string
);`)
	return err
}

// NewMemoryDB opens a private in-memory SQLite database and creates the
// catalog tables. The pool uses one connection so every query shares that
// database. Callers must blank-import github.com/mattn/go-sqlite3.
func NewMemoryDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := Migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
