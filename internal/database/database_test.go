package database

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

func TestMigrateCreatesCatalogOnEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := Migrate(db); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	created, err := NewCategory(db).Create("Tecnologia", "curso de tech")
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	if _, err := NewCourse(db).Create("Full Cycle", "the best", created.ID); err != nil {
		t.Fatalf("create course: %v", err)
	}

	categories, err := NewCategory(db).FindAll()
	if err != nil {
		t.Fatalf("find categories: %v", err)
	}
	if len(categories) != 1 || categories[0].Name != "Tecnologia" {
		t.Fatalf("categories = %+v", categories)
	}
	courses, err := NewCourse(db).FindByCategoryId(created.ID)
	if err != nil {
		t.Fatalf("find courses: %v", err)
	}
	if len(courses) != 1 || courses[0].Name != "Full Cycle" || courses[0].CategoryID != created.ID {
		t.Fatalf("courses = %+v", courses)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := NewMemoryDB()
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestCategoryCreateAndFindAll(t *testing.T) {
	db := openTestDB(t)
	store := NewCategory(db)

	first, err := store.Create("Tecnologia", "curso de tech")
	if err != nil {
		t.Fatalf("create first category: %v", err)
	}
	second, err := store.Create("Design", "")
	if err != nil {
		t.Fatalf("create second category: %v", err)
	}
	if _, err := uuid.Parse(first.ID); err != nil {
		t.Fatalf("first id %q: %v", first.ID, err)
	}
	if first.ID == second.ID {
		t.Fatalf("categories share id %s", first.ID)
	}

	got, err := store.FindAll()
	if err != nil {
		t.Fatalf("find all: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d categories, want 2", len(got))
	}
	byID := map[string]Category{}
	for _, category := range got {
		byID[category.ID] = category
	}
	if byID[first.ID].Name != "Tecnologia" || byID[first.ID].Description != "curso de tech" {
		t.Fatalf("first category = %+v", byID[first.ID])
	}
	if byID[second.ID].Name != "Design" || byID[second.ID].Description != "" {
		t.Fatalf("second category = %+v", byID[second.ID])
	}
}

func TestCategoryFindByCourseID(t *testing.T) {
	db := openTestDB(t)
	categories := NewCategory(db)
	courses := NewCourse(db)

	tech, err := categories.Create("Tecnologia", "curso de tech")
	if err != nil {
		t.Fatalf("create tech: %v", err)
	}
	design, err := categories.Create("Design", "visual")
	if err != nil {
		t.Fatalf("create design: %v", err)
	}
	goCourse, err := courses.Create("Go GraphQL", "gqlgen basics", tech.ID)
	if err != nil {
		t.Fatalf("create go course: %v", err)
	}
	colorCourse, err := courses.Create("Color", "theory", design.ID)
	if err != nil {
		t.Fatalf("create color course: %v", err)
	}

	gotGo, err := categories.FindByCourseId(goCourse.ID)
	if err != nil {
		t.Fatalf("find go category: %v", err)
	}
	if gotGo.ID != tech.ID || gotGo.Name != "Tecnologia" || gotGo.Description != "curso de tech" {
		t.Fatalf("go course category = %+v, want Tecnologia %s", gotGo, tech.ID)
	}

	gotColor, err := categories.FindByCourseId(colorCourse.ID)
	if err != nil {
		t.Fatalf("find color category: %v", err)
	}
	if gotColor.ID != design.ID || gotColor.Name != "Design" {
		t.Fatalf("color course category = %+v, want Design %s", gotColor, design.ID)
	}

	_, err = categories.FindByCourseId("missing-course")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing course error = %v, want sql.ErrNoRows", err)
	}
}

func TestCourseCreateFindAllAndFindByCategoryID(t *testing.T) {
	db := openTestDB(t)
	categories := NewCategory(db)
	courses := NewCourse(db)

	backend, err := categories.Create("Backend", "APIs")
	if err != nil {
		t.Fatalf("create backend: %v", err)
	}
	design, err := categories.Create("Design", "visual")
	if err != nil {
		t.Fatalf("create design: %v", err)
	}

	empty, err := courses.FindAll()
	if err != nil {
		t.Fatalf("find all empty: %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("empty catalog = %#v, want an empty slice", empty)
	}

	goCourse, err := courses.Create("Go GraphQL", "gqlgen basics", backend.ID)
	if err != nil {
		t.Fatalf("create go course: %v", err)
	}
	colorCourse, err := courses.Create("Color", "theory", design.ID)
	if err != nil {
		t.Fatalf("create color course: %v", err)
	}
	// The tables have no foreign key, so an unknown category id is stored.
	orphan, err := courses.Create("Orphan", "no parent", "missing-category")
	if err != nil {
		t.Fatalf("create orphan course: %v", err)
	}

	all, err := courses.FindAll()
	if err != nil {
		t.Fatalf("find all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("got %d courses, want 3", len(all))
	}
	byID := map[string]Course{}
	for _, course := range all {
		byID[course.ID] = course
	}
	if byID[goCourse.ID].Name != "Go GraphQL" || byID[goCourse.ID].CategoryID != backend.ID {
		t.Fatalf("go course = %+v", byID[goCourse.ID])
	}
	if byID[colorCourse.ID].Description != "theory" || byID[colorCourse.ID].CategoryID != design.ID {
		t.Fatalf("color course = %+v", byID[colorCourse.ID])
	}
	if byID[orphan.ID].CategoryID != "missing-category" {
		t.Fatalf("orphan course = %+v", byID[orphan.ID])
	}

	backendCourses, err := courses.FindByCategoryId(backend.ID)
	if err != nil {
		t.Fatalf("find backend courses: %v", err)
	}
	if len(backendCourses) != 1 || backendCourses[0].ID != goCourse.ID || backendCourses[0].CategoryID != backend.ID {
		t.Fatalf("backend courses = %+v", backendCourses)
	}

	none, err := courses.FindByCategoryId("empty-category")
	if err != nil {
		t.Fatalf("find empty category: %v", err)
	}
	if none == nil || len(none) != 0 {
		t.Fatalf("empty category courses = %#v, want an empty slice", none)
	}
}
