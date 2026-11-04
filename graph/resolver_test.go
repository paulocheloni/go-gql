package graph_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/paulocheloni/graphql/graph"
	"github.com/paulocheloni/graphql/graph/model"
	"github.com/paulocheloni/graphql/internal/database"
)

func TestResolversCreateListAndRelate(t *testing.T) {
	ctx := context.Background()
	r := newResolver(t)

	backend, err := r.Mutation().CreateCategory(ctx, model.NewCategory{
		Name:        "Backend",
		Description: strPtr("APIs and services"),
	})
	if err != nil {
		t.Fatalf("create backend: %v", err)
	}
	design, err := r.Mutation().CreateCategory(ctx, model.NewCategory{
		Name:        "Design",
		Description: strPtr("visual"),
	})
	if err != nil {
		t.Fatalf("create design: %v", err)
	}

	goCourse, err := r.Mutation().CreateCourse(ctx, model.NewCourse{
		Name:        "Go GraphQL",
		Description: strPtr("gqlgen basics"),
		CategoryID:  backend.ID,
	})
	if err != nil {
		t.Fatalf("create go course: %v", err)
	}
	if goCourse.Name != "Go GraphQL" || goCourse.Description == nil || *goCourse.Description != "gqlgen basics" {
		t.Fatalf("created course = %+v", goCourse)
	}

	colorCourse, err := r.Mutation().CreateCourse(ctx, model.NewCourse{
		Name:        "Color",
		Description: strPtr("theory"),
		CategoryID:  design.ID,
	})
	if err != nil {
		t.Fatalf("create color course: %v", err)
	}
	services, err := r.Mutation().CreateCourse(ctx, model.NewCourse{
		Name:        "Services",
		Description: strPtr("boundaries"),
		CategoryID:  backend.ID,
	})
	if err != nil {
		t.Fatalf("create services course: %v", err)
	}

	categories, err := r.Query().Categories(ctx)
	if err != nil {
		t.Fatalf("categories: %v", err)
	}
	if len(categories) != 2 {
		t.Fatalf("got %d categories, want 2", len(categories))
	}
	categoryByID := map[string]*model.Category{}
	for _, category := range categories {
		categoryByID[category.ID] = category
	}
	if categoryByID[backend.ID].Description == nil || *categoryByID[backend.ID].Description != "APIs and services" {
		t.Fatalf("listed backend = %+v", categoryByID[backend.ID])
	}
	if categoryByID[design.ID].Description == nil || *categoryByID[design.ID].Description != "visual" {
		t.Fatalf("listed design = %+v", categoryByID[design.ID])
	}

	courses, err := r.Query().Courses(ctx)
	if err != nil {
		t.Fatalf("courses: %v", err)
	}
	if len(courses) != 3 {
		t.Fatalf("got %d courses, want 3", len(courses))
	}
	courseByID := map[string]*model.Course{}
	for _, course := range courses {
		courseByID[course.ID] = course
	}
	if courseByID[goCourse.ID].Description == nil || *courseByID[goCourse.ID].Description != "gqlgen basics" {
		t.Fatalf("listed go course = %+v", courseByID[goCourse.ID])
	}
	if courseByID[colorCourse.ID].Description == nil || *courseByID[colorCourse.ID].Description != "theory" {
		t.Fatalf("listed color course = %+v", courseByID[colorCourse.ID])
	}
	if courseByID[services.ID].Description == nil || *courseByID[services.ID].Description != "boundaries" {
		t.Fatalf("listed services course = %+v", courseByID[services.ID])
	}

	parent, err := r.Course().Category(ctx, goCourse)
	if err != nil {
		t.Fatalf("go course category: %v", err)
	}
	if parent.ID != backend.ID || parent.Name != "Backend" || parent.Description == nil || *parent.Description != "APIs and services" {
		t.Fatalf("go course category = %+v", parent)
	}
	otherParent, err := r.Course().Category(ctx, colorCourse)
	if err != nil {
		t.Fatalf("color course category: %v", err)
	}
	if otherParent.ID != design.ID {
		t.Fatalf("color course category id = %s, want %s", otherParent.ID, design.ID)
	}

	listed, err := r.Category().Courses(ctx, backend)
	if err != nil {
		t.Fatalf("backend courses: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("backend courses = %+v", listed)
	}
	listedByID := map[string]*model.Course{}
	for _, course := range listed {
		listedByID[course.ID] = course
	}
	if listedByID[goCourse.ID] == nil || listedByID[goCourse.ID].Description == nil || *listedByID[goCourse.ID].Description != "gqlgen basics" {
		t.Fatalf("listed go course = %+v", listedByID[goCourse.ID])
	}
	if listedByID[services.ID] == nil || listedByID[services.ID].Description == nil || *listedByID[services.ID].Description != "boundaries" {
		t.Fatalf("listed services course = %+v", listedByID[services.ID])
	}

	// A category with no courses leaves the slice nil. The schema field is
	// nullable, and gqlgen encodes that nil slice as JSON null.
	empty, err := r.Category().Courses(ctx, &model.Category{ID: "unused-category", Name: "Empty"})
	if err != nil {
		t.Fatalf("empty courses: %v", err)
	}
	if empty != nil {
		t.Fatalf("empty courses = %#v, want nil", empty)
	}
}

func TestCreateOmitsOptionalDescription(t *testing.T) {
	ctx := context.Background()
	r := newResolver(t)

	category, err := r.Mutation().CreateCategory(ctx, model.NewCategory{Name: "Untitled"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	if category.Description == nil || *category.Description != "" {
		t.Fatalf("category description = %#v, want empty string", category.Description)
	}

	course, err := r.Mutation().CreateCourse(ctx, model.NewCourse{
		Name:       "Notes",
		CategoryID: category.ID,
	})
	if err != nil {
		t.Fatalf("create course: %v", err)
	}
	if course.Description == nil || *course.Description != "" {
		t.Fatalf("course description = %#v, want empty string", course.Description)
	}

	listed, err := r.Query().Categories(ctx)
	if err != nil {
		t.Fatalf("categories: %v", err)
	}
	if len(listed) != 1 || listed[0].Name != "Untitled" || listed[0].Description == nil || *listed[0].Description != "" {
		t.Fatalf("listed categories = %+v", listed)
	}
}

func TestCourseCategoryMissing(t *testing.T) {
	ctx := context.Background()
	r := newResolver(t)

	course, err := r.Mutation().CreateCourse(ctx, model.NewCourse{
		Name:        "Orphan",
		Description: strPtr("no parent"),
		CategoryID:  "missing-category",
	})
	if err != nil {
		t.Fatalf("create course: %v", err)
	}
	_, err = r.Course().Category(ctx, course)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("category error = %v, want sql.ErrNoRows", err)
	}
}

func TestQueriesReturnDatabaseErrors(t *testing.T) {
	ctx := context.Background()
	db, err := database.NewMemoryDB()
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}
	r := &graph.Resolver{
		CategoryDB: database.NewCategory(db),
		CourseDB:   database.NewCourse(db),
	}

	if _, err := r.Query().Categories(ctx); err == nil {
		t.Fatal("Categories returned nil error for a closed database")
	}
	if _, err := r.Query().Courses(ctx); err == nil {
		t.Fatal("Courses returned nil error for a closed database")
	}
	if _, err := r.Mutation().CreateCategory(ctx, model.NewCategory{Name: "Backend", Description: strPtr("APIs")}); err == nil {
		t.Fatal("CreateCategory returned nil error for a closed database")
	}
}
