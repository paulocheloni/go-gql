package database

import "database/sql"

// NewMemoryDB opens a private in-memory SQLite database and creates the
// catalog tables. The pool uses one connection so every query shares that
// database. Callers must blank-import github.com/mattn/go-sqlite3.
func NewMemoryDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`
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
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
