package graph

import (
	"github.com/paulocheloni/graphql/graph/model"
	"github.com/paulocheloni/graphql/internal/database"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	CategoryDB *database.Category
	CourseDB   *database.Course
}

// stringOrEmpty lets the optional GraphQL description fields be omitted.
func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// newCategoryModel copies description into memory owned by the model.
func newCategoryModel(id, name, description string) *model.Category {
	descriptionCopy := description
	return &model.Category{
		ID:          id,
		Name:        name,
		Description: &descriptionCopy,
	}
}

// newCourseModel copies description into memory owned by the model.
func newCourseModel(id, name, description string) *model.Course {
	descriptionCopy := description
	return &model.Course{
		ID:          id,
		Name:        name,
		Description: &descriptionCopy,
	}
}
