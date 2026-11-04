package graph_test

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/paulocheloni/graphql/graph"
	"github.com/paulocheloni/graphql/internal/database"
)

func newResolver(t *testing.T) *graph.Resolver {
	t.Helper()
	db, err := database.NewMemoryDB()
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &graph.Resolver{
		CategoryDB: database.NewCategory(db),
		CourseDB:   database.NewCourse(db),
	}
}

func strPtr(value string) *string {
	return &value
}
