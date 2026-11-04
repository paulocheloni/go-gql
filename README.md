# go-gql

A small GraphQL API for a course catalog: categories group courses, and a client can list both or create new ones. The service is a single Go process built with [gqlgen](https://github.com/99designs/gqlgen). `graph/schema.graphqls` is the contract, resolvers in `graph/schema.resolvers.go` load and store rows in a SQLite file, and the process also serves gqlgen's GraphQL Playground so the schema can be queried from a browser. It is a portfolio example of a typed GraphQL server: schema, generated runtime, hand-written resolvers, and a thin database layer in one module.

## Tech stack

| Piece | What this repo uses |
| --- | --- |
| Language | Go 1.20 (`go.mod`). Verified with Go 1.22.2. |
| GraphQL | [gqlgen](https://github.com/99designs/gqlgen) v0.17.24 and gqlparser v2.5.1 |
| HTTP | `net/http`. Playground at `/`, API at `/query`. |
| Database | SQLite through `github.com/mattn/go-sqlite3` v1.14.16 (cgo) |
| IDs | UUID strings from `github.com/google/uuid` v1.3.0 |

`gqlgen.yml` binds the GraphQL `Category` and `Course` types to the hand-written structs in `graph/model`. `NewCategory` and `NewCourse` are generated into `graph/model/models_gen.go`. `graph/generated.go` is the gqlgen execution runtime (header: gqlgen v0.17.24). Those generated files are already committed, and `go run` uses them as they are.

## Schema

```graphql
type Category {
  id: ID!
  name: String!
  description: String
  courses: [Course!]
}

type Course {
  id: ID!
  name: String!
  description: String
  category: Category!
}

input NewCategory {
  name: String!
  description: String
}

input NewCourse {
  name: String!
  description: String
  categoryId: ID!
}

type Query {
  categories: [Category!]
  courses: [Course!]
}

type Mutation {
  createCategory(input: NewCategory!): Category!
  createCourse(input: NewCourse!): Course!
}
```

There are two queries and two mutations. `Category.courses` and `Course.category` are field resolvers: the category resolver calls `FindByCategoryId`, and the course resolver calls `FindByCourseId`.

SQLite stores the data in `data.db` in the process working directory. The process opens that file and runs SQL against the tables already in it. The committed schema is:

```sql
CREATE TABLE categories (id string, name string, description string);
CREATE TABLE courses (id string, name string, description string, category_id string);
```

The committed database has one category and two courses:

| Table | id | name | description | category_id |
| --- | --- | --- | --- | --- |
| categories | `b3acefca-a5db-411c-a589-91d4cc7add58` | Tecnologia | curso de tech | |
| courses | `6d3b8b8c-53d9-4501-a41f-0d7d1d6ebb86` | Full Cycle | the best | `b3acefca-a5db-411c-a589-91d4cc7add58` |
| courses | `e68c24b7-c7aa-4593-b5b1-23076efe88bc` | Full Cycle | the best | `b3acefca-a5db-411c-a589-91d4cc7add58` |

### Queries

`categories` returns every category. Request `courses` on a category to load the rows whose `category_id` matches.

```bash
curl -s http://localhost:8080/query \
  -H 'Content-Type: application/json' \
  -d '{"query":"query { categories { id name description courses { id name description } } }"}'
```

Against the committed database that returns:

```json
{
  "data": {
    "categories": [
      {
        "id": "b3acefca-a5db-411c-a589-91d4cc7add58",
        "name": "Tecnologia",
        "description": "curso de tech",
        "courses": [
          {
            "id": "6d3b8b8c-53d9-4501-a41f-0d7d1d6ebb86",
            "name": "Full Cycle",
            "description": "the best"
          },
          {
            "id": "e68c24b7-c7aa-4593-b5b1-23076efe88bc",
            "name": "Full Cycle",
            "description": "the best"
          }
        ]
      }
    ]
  }
}
```

`courses` returns every course. `category` is a nested field on each course:

```bash
curl -s http://localhost:8080/query \
  -H 'Content-Type: application/json' \
  -d '{"query":"query { courses { id name description category { id name description } } }"}'
```

The same query also works as an HTTP GET. gqlgen's default handler accepts both:

```bash
curl -sG http://localhost:8080/query \
  --data-urlencode 'query={ categories { name } }'
```

### Mutations

Send `description` on both creates. The GraphQL type marks it optional, but `CreateCategory` and `CreateCourse` dereference the pointer. Omitting it panics inside the resolver, and gqlgen answers with `{"errors":[{"message":"internal system error","path":["createCategory"]}]}`. An empty string is accepted.

```bash
curl -s http://localhost:8080/query \
  -H 'Content-Type: application/json' \
  -d '{"query":"mutation { createCategory(input: { name: \"Backend\", description: \"APIs and services\" }) { id name description } }"}'
```

```json
{
  "data": {
    "createCategory": {
      "id": "5d8f3ddc-05eb-4244-8d62-476a57bb5724",
      "name": "Backend",
      "description": "APIs and services"
    }
  }
}
```

The `id` above is one value the server generated during a local run. Each call returns a new UUID.

`createCourse` stores `categoryId` on `courses.category_id`. The sample below selects the new course's own fields. Add a `category { ... }` selection on the mutation if you also want the parent category in the same response.

```bash
curl -s http://localhost:8080/query \
  -H 'Content-Type: application/json' \
  -d '{"query":"mutation ($input: NewCourse!) { createCourse(input: $input) { id name description } }","variables":{"input":{"name":"Go GraphQL","description":"gqlgen basics","categoryId":"b3acefca-a5db-411c-a589-91d4cc7add58"}}}'
```

```json
{
  "data": {
    "createCourse": {
      "id": "19c2472b-bc53-449e-9c28-414d9aef5af9",
      "name": "Go GraphQL",
      "description": "gqlgen basics"
    }
  }
}
```

The tables declare no foreign key. A `categoryId` that matches no category is still inserted.

## Project layout

```
.
├── cmd/server.go                 # process entrypoint: SQLite, resolvers, HTTP
├── graph/
│   ├── schema.graphqls           # GraphQL schema
│   ├── schema.resolvers.go       # query, mutation, and relation resolvers
│   ├── resolver.go               # holds *database.Category and *database.Course
│   ├── generated.go              # gqlgen runtime (do not edit by hand)
│   └── model/
│       ├── category.go           # Category struct
│       ├── course.go             # Course struct
│       └── models_gen.go         # generated NewCategory and NewCourse
├── internal/database/
│   ├── category.go               # INSERT and SELECT for categories
│   └── course.go                 # INSERT and SELECT for courses
├── data.db                       # SQLite database, opened as a relative path
├── gqlgen.yml                    # gqlgen layout and model bindings
├── tools.go                      # tools build tag so the module depends on gqlgen
├── go.mod
├── go.sum
└── .env                          # CGO_ENABLED=1; the server does not load this file
```

`cmd/server.go` opens `data.db`, builds `database.Category` and `database.Course`, and passes them to `graph.NewExecutableSchema`. `GET /` is `playground.Handler` titled "GraphQL playground". `POST` and `GET /query` are `handler.NewDefaultServer`. The listen port is `PORT`, or `8080` when that variable is empty.

## Run locally

Requirements: Go 1.20 or newer, and a C compiler. `mattn/go-sqlite3` builds with cgo. `.env` contains `CGO_ENABLED=1` for that reason. The server never reads `.env`; export the variable in the shell when your environment has cgo turned off (common on Windows).

From the repository root, so the relative `data.db` path resolves to the committed database:

```bash
export CGO_ENABLED=1
go run ./cmd/server.go
```

The process logs `connect to http://localhost:8080/ for GraphQL playground`. Open that URL for the playground, which posts to `/query`. To use another port:

```bash
PORT=9090 go run ./cmd/server.go
```

A local run of `go run ./cmd/server.go` with `CGO_ENABLED=1` served the playground (`HTTP 200`) and answered the list and create operations shown above.

Creating rows writes into `data.db`. Copy the file first if you want to keep the seed data.

## Tests

The module has no `*_test.go` files. The Go test command still runs, and each package reports that it has nothing to execute:

```bash
export CGO_ENABLED=1
go test ./...
```

Verified output (exit code 0):

```
?   	github.com/paulocheloni/graphql/cmd	[no test files]
?   	github.com/paulocheloni/graphql/graph	[no test files]
?   	github.com/paulocheloni/graphql/graph/model	[no test files]
?   	github.com/paulocheloni/graphql/internal/database	[no test files]
```

## Resolver behavior worth knowing

Two details in the current resolvers change what a client should expect:

- `Category.courses` selects `WHERE category_id = $1`. A category with no courses is returned as `"courses": null`, because the resolver leaves the result slice nil when the query returns no rows. The list type in the schema is nullable, so that response validates.
- `Course.category` runs `SELECT c.id, c.name, c.description FROM categories AS c JOIN courses co ON c.id = co.category_id`. The course id argument is unused: the statement has no placeholder, and `QueryRow` returns the first joined category. In the committed database every course belongs to Tecnologia, which is the row this field returns.
