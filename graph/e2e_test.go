package graph_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/paulocheloni/graphql/graph"
)

func TestGraphQLQueryAndMutation(t *testing.T) {
	resolver := newResolver(t)
	srv := httptest.NewServer(handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver})))
	t.Cleanup(srv.Close)

	created := postGraphQL(t, srv.URL, `mutation ($input: NewCategory!) {
		createCategory(input: $input) { id name description }
	}`, map[string]any{
		"input": map[string]any{"name": "Backend", "description": "APIs and services"},
	})
	category := dataField(t, created, "createCategory")
	categoryID, _ := category["id"].(string)
	if categoryID == "" || category["name"] != "Backend" || category["description"] != "APIs and services" {
		t.Fatalf("createCategory = %#v", category)
	}

	omitted := postGraphQL(t, srv.URL, `mutation {
		createCategory(input: { name: "Untitled" }) { id name description }
	}`, nil)
	untitled := dataField(t, omitted, "createCategory")
	if untitled["name"] != "Untitled" || untitled["description"] != "" {
		t.Fatalf("omitted description = %#v", untitled)
	}

	courseResp := postGraphQL(t, srv.URL, `mutation ($input: NewCourse!) {
		createCourse(input: $input) { id name description }
	}`, map[string]any{
		"input": map[string]any{
			"name":        "Go GraphQL",
			"description": "gqlgen basics",
			"categoryId":  categoryID,
		},
	})
	course := dataField(t, courseResp, "createCourse")
	courseID, _ := course["id"].(string)
	if courseID == "" || course["name"] != "Go GraphQL" || course["description"] != "gqlgen basics" {
		t.Fatalf("createCourse = %#v", course)
	}

	listed := postGraphQL(t, srv.URL, `query {
		courses { id name description category { id name description } }
	}`, nil)
	courses := dataList(t, listed, "courses")
	if len(courses) != 1 {
		t.Fatalf("courses = %#v", courses)
	}
	if courses[0]["id"] != courseID {
		t.Fatalf("course id = %#v, want %s", courses[0]["id"], courseID)
	}
	parent, _ := courses[0]["category"].(map[string]any)
	if parent["id"] != categoryID || parent["name"] != "Backend" || parent["description"] != "APIs and services" {
		t.Fatalf("course category = %#v", parent)
	}

	body := getGraphQL(t, srv.URL, `{ categories { name description courses { name } } }`)
	var got struct {
		Data struct {
			Categories []struct {
				Name        string          `json:"name"`
				Description string          `json:"description"`
				Courses     json.RawMessage `json:"courses"`
			} `json:"categories"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode categories: %v\n%s", err, body)
	}
	if len(got.Errors) != 0 {
		t.Fatalf("categories errors = %+v", got.Errors)
	}
	if len(got.Data.Categories) != 2 {
		t.Fatalf("categories = %+v", got.Data.Categories)
	}
	byName := map[string]struct {
		Description string
		Courses     string
		CourseName  string
	}{}
	for _, category := range got.Data.Categories {
		entry := byName[category.Name]
		entry.Description = category.Description
		entry.Courses = string(category.Courses)
		if entry.Courses != "null" {
			var courses []struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(category.Courses, &courses); err != nil {
				t.Fatalf("decode %s courses: %v", category.Name, err)
			}
			if len(courses) == 1 {
				entry.CourseName = courses[0].Name
			}
		}
		byName[category.Name] = entry
	}
	if byName["Backend"].CourseName != "Go GraphQL" || byName["Backend"].Description != "APIs and services" {
		t.Fatalf("backend = %+v", byName["Backend"])
	}
	// A category with no courses is a nil slice, and gqlgen encodes that as JSON null.
	if byName["Untitled"].Courses != "null" || byName["Untitled"].Description != "" {
		t.Fatalf("untitled = %+v", byName["Untitled"])
	}

	invalid := postGraphQLExpect(t, srv.URL, `mutation { createCategory(input: { description: "missing name" }) { id } }`, nil, http.StatusUnprocessableEntity)
	if len(invalid.Errors) == 0 || invalid.Errors[0].Message == "" {
		t.Fatalf("missing name response = %#v", invalid)
	}
}

type graphQLResponse struct {
	Data   map[string]any `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func postGraphQL(t *testing.T, endpoint, query string, variables map[string]any) graphQLResponse {
	t.Helper()
	return postGraphQLExpect(t, endpoint, query, variables, http.StatusOK)
}

func postGraphQLExpect(t *testing.T, endpoint, query string, variables map[string]any, status int) graphQLResponse {
	t.Helper()
	payload := map[string]any{"query": query}
	if variables != nil {
		payload["variables"] = variables
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	res, err := http.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if res.StatusCode != status {
		t.Fatalf("status %d, want %d: %s", res.StatusCode, status, raw)
	}
	var decoded graphQLResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode: %v\n%s", err, raw)
	}
	return decoded
}

func getGraphQL(t *testing.T, endpoint, query string) []byte {
	t.Helper()
	res, err := http.Get(endpoint + "?query=" + url.QueryEscape(query))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("status %d: %s", res.StatusCode, raw)
	}
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return raw
}

func dataField(t *testing.T, res graphQLResponse, field string) map[string]any {
	t.Helper()
	if len(res.Errors) != 0 {
		t.Fatalf("graphql errors = %+v", res.Errors)
	}
	value, ok := res.Data[field].(map[string]any)
	if !ok {
		t.Fatalf("data.%s = %#v", field, res.Data[field])
	}
	return value
}

func dataList(t *testing.T, res graphQLResponse, field string) []map[string]any {
	t.Helper()
	if len(res.Errors) != 0 {
		t.Fatalf("graphql errors = %+v", res.Errors)
	}
	raw, ok := res.Data[field].([]any)
	if !ok {
		t.Fatalf("data.%s = %#v", field, res.Data[field])
	}
	items := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		typed, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("list item %#v", item)
		}
		items = append(items, typed)
	}
	return items
}
