# BACKEND_GRAPHQL.md

GraphQL API design and implementation guide for the `charging-backend` application.

This document covers the common pagination/filter framework shared by all list endpoints,
and the first concrete resource implementation: **CarrierResource**.

The design is a port of the Java/Quarkus/Panache solution. The Go implementation:
- Uses **gqlgen** for GraphQL code generation
- Uses **sqlc** for static, known-at-compile-time SQL queries
- Uses **direct pgx** for dynamic (filtered/paginated) queries that cannot be expressed statically
- Follows the established repository pattern — all database access through `internal/store/`

---

## Table of Contents

1. [Common Input Types](#1-common-input-types)
2. [Filter Builder Package](#2-filter-builder-package)
3. [Carrier GraphQL Schema](#3-carrier-graphql-schema)
4. [Schema Root Updates](#4-schema-root-updates)
5. [SQL Query Additions](#5-sql-query-additions)
6. [Store Layer — Dynamic Queries](#6-store-layer--dynamic-queries)
7. [Service Layer](#7-service-layer)
8. [Resolver Layer](#8-resolver-layer)
9. [AppContext Wiring](#9-appcontext-wiring)
10. [gqlgen.yml Updates](#10-gqlgenyml-updates)
11. [File Map](#11-file-map)
12. [Generation Commands](#12-generation-commands)

---

## 1. Common Input Types

These types are shared across **all** GraphQL list endpoints. They are defined once in
`gql/schema/schema.graphql` so every resource can reuse them without repetition.

### GraphQL Schema Fragment

Add the following to `gql/schema/schema.graphql` below the existing scalar declarations:

```graphql
# Pagination control — mirrors Java PageRequest record.
# pageIndex defaults to 0 if null or negative.
# pageSize  defaults to 10 if null or zero.
input PageRequest {
  pageIndex: Int
  pageSize:  Int
  sort:      SortInput
}

# Sort direction for list queries — mirrors Java SortRequest record.
# order must be "asc" or "desc"; any other value is treated as "asc".
input SortInput {
  key:   String!
  order: String!
}

# A single field-level predicate — mirrors Java Filter record.
# Allowed operations: =  <>  <  >  <=  >=  ILIKE  LIKE  IN  NOT IN
# For IN and NOT IN, supply a comma-separated value e.g. "NZL,AUS,GBR".
input FilterInput {
  key:       String!
  operation: String!
  value:     String!
}

# Bundle of predicates and an optional wildcard term — mirrors Java FilterRequest.
# All entries in filters are ANDed together.
# wildcard is searched across resource-defined columns using ILIKE.
input FilterRequest {
  filters:  [FilterInput!]
  wildcard: String
}
```

### Semantics

| Java concept | Go/GraphQL equivalent |
|---|---|
| `PageRequest.pageIndex` | `PageRequest.pageIndex: Int` — defaults to 0 |
| `PageRequest.pageSize` | `PageRequest.pageSize: Int` — defaults to 10 |
| `SortRequest.key` blank → use default column | `SortInput.key` empty → falls back to resource default |
| `SortRequest.order "asc"` → Ascending | `SortInput.order "asc"` → ASC; anything else → ASC |
| `FilterRequest.filters` JPQL named params | `FilterRequest.filters` → SQL `$N` positional params |
| `FilterRequest.wildcard` → `ILIKE :_wildcard` across wildcard fields | `FilterRequest.wildcard` → `ILIKE $N` across resource wildcard columns |
| `Filter.operation "IN"/"NOT IN"` → `value.split(",")` | Same — value split on comma into individual `$N` args |

---

## 2. Filter Builder Package

**Location:** `internal/backend/filter/filter.go`

This package is the Go equivalent of Java's `FilterRequest.buildFilter()`. It translates
the GraphQL input types into parameterised SQL fragments. A per-resource column allowlist
is the **only** mechanism that prevents SQL injection — user-supplied strings never appear
in the query text, only in the argument list.

### Design

- `BuildWhere` produces a `WhereClause{SQL, Args}` from a `*model.FilterRequest`.
- `BuildOrderBy` produces an `ORDER BY` string from a `*model.PageRequest`.
- `PageOffset` extracts `(limit, offset int)` from a `*model.PageRequest`, applying defaults.
- Each resource provides two `map[string]string` values: one for filter/sort column mapping
  (GraphQL camelCase field name → SQL snake_case column name) and one slice of SQL column
  names for the wildcard search.

### Full Package Code

```go
// Package filter provides generic SQL WHERE clause and ORDER BY builders for
// GraphQL list endpoints. It is the Go equivalent of Java FilterRequest.buildFilter().
package filter

import (
	"fmt"
	"strings"

	"go-ocs/internal/backend/graphql/model"
)

// WhereClause holds a SQL WHERE fragment and its positional bound arguments.
// SQL is empty string when no filters were specified (no WHERE keyword is emitted).
type WhereClause struct {
	SQL  string // e.g. "WHERE carrier_name ILIKE $1 AND iso = $2"
	Args []any
}

// allowedOperations is the whitelist of permitted SQL comparison operators.
// User-supplied operation strings are upper-cased before lookup.
var allowedOperations = map[string]bool{
	"=": true, "<>": true, "!=": true,
	"<": true, ">": true, "<=": true, ">=": true,
	"ILIKE": true, "LIKE": true,
	"IN": true, "NOT IN": true,
}

// BuildWhere constructs a parameterised SQL WHERE clause from a FilterRequest.
//
// allowedCols maps GraphQL field names (camelCase) to SQL column names (snake_case).
// Any key not present in allowedCols is rejected with an error, preventing SQL injection.
//
// wildcardCols are the SQL column names searched when req.Wildcard is set.
// They are ORed together inside parentheses, then ANDed with the other predicates.
func BuildWhere(
	req *model.FilterRequest,
	allowedCols map[string]string,
	wildcardCols []string,
) (WhereClause, error) {
	if req == nil {
		return WhereClause{}, nil
	}

	var clauses []string
	var args []any
	idx := 1

	for _, f := range req.Filters {
		col, ok := allowedCols[f.Key]
		if !ok {
			return WhereClause{}, fmt.Errorf("filter key %q is not permitted", f.Key)
		}

		op := strings.ToUpper(strings.TrimSpace(f.Operation))
		if !allowedOperations[op] {
			return WhereClause{}, fmt.Errorf("filter operation %q is not permitted", f.Operation)
		}

		if op == "IN" || op == "NOT IN" {
			// Split comma-separated value into individual positional args
			parts := strings.Split(f.Value, ",")
			placeholders := make([]string, 0, len(parts))
			for _, p := range parts {
				args = append(args, strings.TrimSpace(p))
				placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
				idx++
			}
			clauses = append(clauses,
				fmt.Sprintf("%s %s (%s)", col, op, strings.Join(placeholders, ", ")))
		} else {
			clauses = append(clauses, fmt.Sprintf("%s %s $%d", col, f.Operation, idx))
			args = append(args, f.Value)
			idx++
		}
	}

	// Wildcard: searched across all wildcard columns using ILIKE, ORed together
	if req.Wildcard != nil && strings.TrimSpace(*req.Wildcard) != "" {
		wcClauses := make([]string, 0, len(wildcardCols))
		for _, col := range wildcardCols {
			wcClauses = append(wcClauses, fmt.Sprintf("%s ILIKE $%d", col, idx))
		}
		if len(wcClauses) > 0 {
			clauses = append(clauses, "("+strings.Join(wcClauses, " OR ")+")")
			args = append(args, "%"+strings.TrimSpace(*req.Wildcard)+"%")
			idx++
		}
	}

	if len(clauses) == 0 {
		return WhereClause{}, nil
	}
	return WhereClause{
		SQL:  "WHERE " + strings.Join(clauses, " AND "),
		Args: args,
	}, nil
}

// BuildOrderBy constructs a SQL ORDER BY clause from the sort fields in a PageRequest.
//
// defaultCol is used when no sort key is specified or the request is nil.
// allowedCols is the same per-resource map used by BuildWhere.
func BuildOrderBy(
	req *model.PageRequest,
	defaultCol string,
	allowedCols map[string]string,
) (string, error) {
	col := defaultCol
	dir := "ASC"

	if req != nil && req.Sort != nil && strings.TrimSpace(req.Sort.Key) != "" {
		mapped, ok := allowedCols[req.Sort.Key]
		if !ok {
			return "", fmt.Errorf("sort key %q is not permitted", req.Sort.Key)
		}
		col = mapped
		if strings.EqualFold(strings.TrimSpace(req.Sort.Order), "desc") {
			dir = "DESC"
		}
	}

	return fmt.Sprintf("ORDER BY %s %s", col, dir), nil
}

// PageOffset extracts LIMIT and OFFSET values from a PageRequest, applying safe defaults:
// pageIndex defaults to 0, pageSize defaults to 10.
func PageOffset(req *model.PageRequest) (limit, offset int) {
	pageIndex := 0
	pageSize := 10

	if req != nil {
		if req.PageIndex != nil && *req.PageIndex > 0 {
			pageIndex = int(*req.PageIndex)
		}
		if req.PageSize != nil && *req.PageSize > 0 {
			pageSize = int(*req.PageSize)
		}
	}
	return pageSize, pageIndex * pageSize
}
```

> **Note on gqlgen types:** After running `gqlgen generate`, `model.PageRequest.PageIndex`
> and `model.PageRequest.PageSize` are `*int32` (gqlgen maps nullable `Int` to `*int32`).
> The `int()` cast in `PageOffset` handles this. If you prefer `int` throughout, add a
> models binding in `gqlgen.yml` (see [Section 10](#10-gqlgenyml-updates)).

---

## 3. Carrier GraphQL Schema

**File:** `gql/schema/charging.graphql` — **full replacement**

```graphql
# Carrier represents a wholesale carrier / mobile network operator.
# Maps directly to the charging.carrier database table.
type Carrier {
  plmn:             String!   # Primary key — MCC+MNC combined e.g. "53005"
  mcc:              String!   # Mobile Country Code (3 digits)
  mnc:              String    # Mobile Network Code (up to 3 digits, optional)
  carrierName:      String!
  sourceGroup:      String!   # Roaming source classification (maps to RateKey.SOURCE_TYPE)
  destinationGroup: String!   # Service destination group (maps to RateKey.SERVICE_CATEGORY)
  countryName:      String!
  iso:              String!   # ISO 3166-1 alpha-3 country code e.g. "NZL"
  modifiedOn:       DateTime  # Last modified timestamp — managed by the database; read-only
}

# CarrierInput is used for both createCarrier and updateCarrier mutations.
# On updateCarrier the plmn path argument is authoritative;
# the plmn field inside this input is ignored for the WHERE predicate.
input CarrierInput {
  plmn:             String!
  mcc:              String!
  mnc:              String
  carrierName:      String!
  sourceGroup:      String!
  destinationGroup: String!
  countryName:      String!
  iso:              String!
}

extend type Query {
  # Returns a filtered, sorted, paginated list of carriers.
  carrierList(page: PageRequest, filter: FilterRequest): [Carrier!]!

  # Returns a single carrier by PLMN, or null if not found.
  carrierByPlmn(plmn: String!): Carrier

  # Returns the total count of carriers matching the supplied filter.
  countCarriers(filter: FilterRequest): Int!
}

extend type Mutation {
  # Creates a new carrier and returns the persisted record (including modifiedOn).
  createCarrier(carrier: CarrierInput!): Carrier!

  # Updates an existing carrier by PLMN and returns the updated record.
  updateCarrier(plmn: String!, carrier: CarrierInput!): Carrier!

  # Deletes a carrier by PLMN. Returns true on success.
  deleteCarrier(plmn: String!): Boolean!
}
```

---

## 4. Schema Root Updates

**File:** `gql/schema/schema.graphql` — add the four common input types defined in
[Section 1](#1-common-input-types) below the existing `scalar DateTime` line.
The `type Query`, `type Mutation`, `scalar DateTime`, and `type AdminUser` blocks remain unchanged.

---

## 5. SQL Query Additions

**File:** `internal/store/queries/carrriers.sql` — append to the existing file.

The existing `AllCarriers` query is retained. The four new queries below cover the static
CRUD operations. The dynamic `carrierList` and `countCarriers` queries are **not** expressed
here because their WHERE and ORDER BY clauses are runtime-constructed; see
[Section 6](#6-store-layer--dynamic-queries).

```sql
-- name: CarrierByPlmn :one
-- Retrieves a single carrier record by its PLMN identifier.
SELECT plmn, modified_on, mcc, mnc, carrier_name, source_group,
       destination_group, country_name, iso
FROM carrier
WHERE plmn = $1;

-- name: CreateCarrier :one
-- Inserts a new carrier and returns the full persisted row including modified_on.
INSERT INTO carrier (
    plmn, mcc, mnc, carrier_name, source_group,
    destination_group, country_name, iso, modified_on
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, NOW()
) RETURNING plmn, modified_on, mcc, mnc, carrier_name, source_group,
            destination_group, country_name, iso;

-- name: UpdateCarrier :one
-- Updates an existing carrier by PLMN and returns the updated row.
-- modified_on is refreshed to NOW() on every update.
UPDATE carrier
SET mcc               = $2,
    mnc               = $3,
    carrier_name      = $4,
    source_group      = $5,
    destination_group = $6,
    country_name      = $7,
    iso               = $8,
    modified_on       = NOW()
WHERE plmn = $1
RETURNING plmn, modified_on, mcc, mnc, carrier_name, source_group,
          destination_group, country_name, iso;

-- name: DeleteCarrier :exec
-- Deletes a carrier by PLMN.
DELETE FROM carrier
WHERE plmn = $1;
```

> **Note on `query_parameter_limit: 4`:** sqlc is configured with a parameter limit of 4.
> `CreateCarrier` (8 params) and `UpdateCarrier` (8 params) will therefore generate
> `CreateCarrierParams` and `UpdateCarrierParams` structs automatically. After appending
> these queries, run `sqlc generate` to regenerate `internal/store/sqlc/carrriers.sql.go`.

---

## 6. Store Layer — Dynamic Queries

**File:** `internal/store/carrier_store.go` — new hand-written file (not sqlc-generated).

This file adds two methods to the existing `Store` struct to support the dynamic
`carrierList` and `countCarriers` operations. All SQL is parameterised; the WHERE and
ORDER BY fragments are constructed by the filter builder and passed in — no string
concatenation of user data ever occurs here.

```go
package store

import (
	"context"
	"fmt"

	"go-ocs/internal/store/sqlc"
)

// ListCarriersParams holds the runtime-constructed SQL fragments for a dynamic carrier query.
// WhereSQL and Args come from filter.BuildWhere; OrderSQL from filter.BuildOrderBy.
type ListCarriersParams struct {
	WhereSQL string // e.g. "WHERE carrier_name ILIKE $1"  (empty string = no filter)
	Args     []any  // positional args for the WHERE clause
	OrderSQL string // e.g. "ORDER BY plmn ASC"
	Limit    int
	Offset   int
}

// ListCarriers executes a dynamic carrier query with optional filtering, sorting,
// and pagination. LIMIT and OFFSET are appended as the final positional arguments.
func (s *Store) ListCarriers(ctx context.Context, p ListCarriersParams) ([]sqlc.Carrier, error) {
	limitIdx := len(p.Args) + 1
	offsetIdx := limitIdx + 1

	q := fmt.Sprintf(
		`SELECT plmn, modified_on, mcc, mnc, carrier_name, source_group,
		        destination_group, country_name, iso
		 FROM carrier %s %s LIMIT $%d OFFSET $%d`,
		p.WhereSQL, p.OrderSQL, limitIdx, offsetIdx,
	)
	args := append(p.Args, p.Limit, p.Offset)

	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var carriers []sqlc.Carrier
	for rows.Next() {
		var c sqlc.Carrier
		if err := rows.Scan(
			&c.Plmn, &c.ModifiedOn, &c.Mcc, &c.Mnc,
			&c.CarrierName, &c.SourceGroup, &c.DestinationGroup,
			&c.CountryName, &c.Iso,
		); err != nil {
			return nil, err
		}
		carriers = append(carriers, c)
	}
	return carriers, rows.Err()
}

// CountCarriers executes a dynamic count query with optional filtering.
// whereSQL and args are produced by filter.BuildWhere.
func (s *Store) CountCarriers(ctx context.Context, whereSQL string, args []any) (int64, error) {
	q := fmt.Sprintf("SELECT COUNT(*) FROM carrier %s", whereSQL)
	var n int64
	if err := s.DB.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
```

---

## 7. Service Layer

**File:** `internal/backend/services/carrier_service.go` — new file.

The service owns all carrier business logic. It composes the filter builder output with
the store calls, and maps between `sqlc.Carrier` (database model) and `model.Carrier`
(GraphQL model). Handlers and resolvers must never call the store directly.

```go
package services

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"go-ocs/internal/backend/filter"
	"go-ocs/internal/backend/graphql/model"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// carrierColumns maps GraphQL field names to SQL column names for the carrier table.
// Only fields present in this map are accepted as filter or sort keys.
var carrierColumns = map[string]string{
	"plmn":             "plmn",
	"mcc":              "mcc",
	"mnc":              "mnc",
	"carrierName":      "carrier_name",
	"sourceGroup":      "source_group",
	"destinationGroup": "destination_group",
	"countryName":      "country_name",
	"iso":              "iso",
	"modifiedOn":       "modified_on",
}

// carrierWildcardCols are the SQL column names searched when a wildcard term is provided.
// Mirrors Java's CarrierEntity.WILDCARD_FIELDS constant.
var carrierWildcardCols = []string{
	"plmn", "carrier_name", "country_name", "iso", "source_group", "destination_group",
}

// CarrierService handles all business logic for the carrier resource.
type CarrierService struct {
	store *store.Store
}

// NewCarrierService creates a new CarrierService backed by the given store.
func NewCarrierService(s *store.Store) *CarrierService {
	return &CarrierService{store: s}
}

// ListCarriers returns a filtered, sorted, paginated list of carriers.
func (s *CarrierService) ListCarriers(
	ctx context.Context,
	page *model.PageRequest,
	filterReq *model.FilterRequest,
) ([]*model.Carrier, error) {
	where, err := filter.BuildWhere(filterReq, carrierColumns, carrierWildcardCols)
	if err != nil {
		return nil, err
	}
	orderBy, err := filter.BuildOrderBy(page, "plmn", carrierColumns)
	if err != nil {
		return nil, err
	}
	limit, offset := filter.PageOffset(page)

	rows, err := s.store.ListCarriers(ctx, store.ListCarriersParams{
		WhereSQL: where.SQL,
		Args:     where.Args,
		OrderSQL: orderBy,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*model.Carrier, 0, len(rows))
	for _, c := range rows {
		result = append(result, carrierToModel(c))
	}
	return result, nil
}

// CarrierByPlmn returns a single carrier by PLMN, or nil if not found.
func (s *CarrierService) CarrierByPlmn(ctx context.Context, plmn string) (*model.Carrier, error) {
	c, err := s.store.Q.CarrierByPlmn(ctx, plmn)
	if err != nil {
		return nil, err
	}
	return carrierToModel(c), nil
}

// CountCarriers returns the total count of carriers matching the supplied filter.
func (s *CarrierService) CountCarriers(
	ctx context.Context,
	filterReq *model.FilterRequest,
) (int, error) {
	where, err := filter.BuildWhere(filterReq, carrierColumns, carrierWildcardCols)
	if err != nil {
		return 0, err
	}
	n, err := s.store.CountCarriers(ctx, where.SQL, where.Args)
	return int(n), err
}

// CreateCarrier persists a new carrier and returns the created record.
func (s *CarrierService) CreateCarrier(
	ctx context.Context,
	input model.CarrierInput,
) (*model.Carrier, error) {
	c, err := s.store.Q.CreateCarrier(ctx, sqlc.CreateCarrierParams{
		Plmn:             input.Plmn,
		Mcc:              input.Mcc,
		Mnc:              textFromPtr(input.Mnc),
		CarrierName:      input.CarrierName,
		SourceGroup:      input.SourceGroup,
		DestinationGroup: input.DestinationGroup,
		CountryName:      input.CountryName,
		Iso:              input.Iso,
	})
	if err != nil {
		return nil, err
	}
	return carrierToModel(c), nil
}

// UpdateCarrier modifies an existing carrier by PLMN and returns the updated record.
// The plmn argument is authoritative; input.Plmn is ignored.
func (s *CarrierService) UpdateCarrier(
	ctx context.Context,
	plmn string,
	input model.CarrierInput,
) (*model.Carrier, error) {
	c, err := s.store.Q.UpdateCarrier(ctx, sqlc.UpdateCarrierParams{
		Plmn:             plmn,
		Mcc:              input.Mcc,
		Mnc:              textFromPtr(input.Mnc),
		CarrierName:      input.CarrierName,
		SourceGroup:      input.SourceGroup,
		DestinationGroup: input.DestinationGroup,
		CountryName:      input.CountryName,
		Iso:              input.Iso,
	})
	if err != nil {
		return nil, fmt.Errorf("update carrier %s: %w", plmn, err)
	}
	return carrierToModel(c), nil
}

// DeleteCarrier removes a carrier by PLMN. Returns true on success.
func (s *CarrierService) DeleteCarrier(ctx context.Context, plmn string) (bool, error) {
	if err := s.store.Q.DeleteCarrier(ctx, plmn); err != nil {
		return false, fmt.Errorf("delete carrier %s: %w", plmn, err)
	}
	return true, nil
}

// carrierToModel maps a sqlc.Carrier row to the GraphQL model type.
func carrierToModel(c sqlc.Carrier) *model.Carrier {
	m := &model.Carrier{
		Plmn:             c.Plmn,
		Mcc:              c.Mcc,
		CarrierName:      c.CarrierName,
		SourceGroup:      c.SourceGroup,
		DestinationGroup: c.DestinationGroup,
		CountryName:      c.CountryName,
		Iso:              c.Iso,
	}
	if c.Mnc.Valid {
		m.Mnc = &c.Mnc.String
	}
	if c.ModifiedOn.Valid {
		t := c.ModifiedOn.Time
		m.ModifiedOn = &t
	}
	return m
}

// textFromPtr converts a nullable *string to a pgtype.Text for sqlc parameters.
func textFromPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}
```

---

## 8. Resolver Layer

**File:** `internal/backend/resolvers/charging.resolvers.go`

gqlgen generates a skeleton file for each schema file. When you run `gqlgen generate`
after updating `charging.graphql`, it will create `charging.resolvers.go` with the correct
method signatures. Fill in the bodies as shown below — each method delegates entirely to
the service; resolvers contain no business logic.

```go
package resolvers

// This file is generated by gqlgen. Resolver bodies are filled in manually.
// DO NOT remove or rename the methods — gqlgen will regenerate them on the next run.

import (
	"context"

	"go-ocs/internal/backend/graphql/model"
)

// --- Queries ---

// CarrierList returns a filtered, sorted, paginated list of carriers.
func (r *queryResolver) CarrierList(
	ctx context.Context,
	page *model.PageRequest,
	filter *model.FilterRequest,
) ([]*model.Carrier, error) {
	return r.carrierSvc.ListCarriers(ctx, page, filter)
}

// CarrierByPlmn returns a single carrier by PLMN, or null if not found.
func (r *queryResolver) CarrierByPlmn(
	ctx context.Context,
	plmn string,
) (*model.Carrier, error) {
	return r.carrierSvc.CarrierByPlmn(ctx, plmn)
}

// CountCarriers returns the total number of carriers matching the filter.
func (r *queryResolver) CountCarriers(
	ctx context.Context,
	filter *model.FilterRequest,
) (int, error) {
	return r.carrierSvc.CountCarriers(ctx, filter)
}

// --- Mutations ---

// CreateCarrier persists a new carrier and returns the created record.
func (r *mutationResolver) CreateCarrier(
	ctx context.Context,
	carrier model.CarrierInput,
) (*model.Carrier, error) {
	return r.carrierSvc.CreateCarrier(ctx, carrier)
}

// UpdateCarrier modifies an existing carrier and returns the updated record.
func (r *mutationResolver) UpdateCarrier(
	ctx context.Context,
	plmn string,
	carrier model.CarrierInput,
) (*model.Carrier, error) {
	return r.carrierSvc.UpdateCarrier(ctx, plmn, carrier)
}

// DeleteCarrier removes a carrier by PLMN. Returns true on success.
func (r *mutationResolver) DeleteCarrier(
	ctx context.Context,
	plmn string,
) (bool, error) {
	return r.carrierSvc.DeleteCarrier(ctx, plmn)
}
```

**File:** `internal/backend/resolvers/resolver.go` — gqlgen also generates this root file.
Add service dependencies to the `Resolver` struct:

```go
package resolvers

import "go-ocs/internal/backend/services"

// Resolver is the root dependency container injected into all generated resolvers.
type Resolver struct {
	carrierSvc *services.CarrierService
}

// NewResolver creates the root Resolver with all service dependencies wired in.
func NewResolver(carrierSvc *services.CarrierService) *Resolver {
	return &Resolver{carrierSvc: carrierSvc}
}

type queryResolver    struct{ *Resolver }
type mutationResolver struct{ *Resolver }

// Query returns the query resolver — required by the gqlgen generated interface.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

// Mutation returns the mutation resolver — required by the gqlgen generated interface.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }
```

---

## 9. AppContext Wiring

**File:** `internal/backend/appcontext/context.go` — add `CarrierService` to `AppContext`.

```go
// Add to the AppContext struct:
CarrierService *services.CarrierService

// Add to the NewAppContext constructor (after store is initialised):
CarrierService: services.NewCarrierService(store),
```

**File:** `internal/backend/handlers/graphql/router.go` — wire the gqlgen handler.

```go
// Replace the placeholder comment with the real gqlgen handler:
import (
    "github.com/99designs/gqlgen/graphql/handler"
    "go-ocs/internal/backend/graphql/generated"
    "go-ocs/internal/backend/resolvers"
)

resolver := resolvers.NewResolver(appCtx.CarrierService)
srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
    Resolvers: resolver,
}))
r.Handle("/", srv)
```

---

## 10. gqlgen.yml Updates

The only required change is a `models` section to map GraphQL `Int` to Go `int` (rather
than the default `int32`). This keeps the service and resolver signatures consistent with
the rest of the codebase which uses `int`.

```yaml
# gql/gqlgen.yml — add below the resolver section:
models:
  Int:
    model:
      - github.com/99designs/gqlgen/graphql.Int
  Int64:
    model:
      - github.com/99designs/gqlgen/graphql.Int64
```

With this binding, `PageRequest.PageIndex` and `PageRequest.PageSize` will be `*int`
(nullable `Int`) and `countCarriers` will return `int` — no int32/int64 casts needed.

---

## 11. File Map

| File | Action | Notes |
|---|---|---|
| `gql/schema/schema.graphql` | **Modify** | Add `PageRequest`, `SortInput`, `FilterInput`, `FilterRequest` input types |
| `gql/schema/charging.graphql` | **Replace** | Full `Carrier` type + `CarrierInput` + `extend type Query/Mutation` |
| `gql/gqlgen.yml` | **Modify** | Add `models:` section to bind `Int` → `int` |
| `internal/store/queries/carrriers.sql` | **Append** | `CarrierByPlmn`, `CreateCarrier`, `UpdateCarrier`, `DeleteCarrier` |
| `internal/store/sqlc/carrriers.sql.go` | **Regenerate** | Run `sqlc generate` after SQL changes |
| `internal/store/carrier_store.go` | **New** | `ListCarriers()` and `CountCarriers()` dynamic pgx methods on `Store` |
| `internal/backend/filter/filter.go` | **New** | `BuildWhere()`, `BuildOrderBy()`, `PageOffset()` — shared utility |
| `internal/backend/services/carrier_service.go` | **New** | `CarrierService` — all six operations + model mapping |
| `internal/backend/resolvers/resolver.go` | **New** | `Resolver` root struct with service dependencies |
| `internal/backend/resolvers/charging.resolvers.go` | **New (gqlgen skeleton + bodies)** | Thin resolver methods delegating to `CarrierService` |
| `internal/backend/graphql/generated/generated.go` | **Regenerate** | Run `gqlgen generate` after schema changes |
| `internal/backend/graphql/model/models_gen.go` | **Regenerate** | Run `gqlgen generate` — produces `Carrier`, `CarrierInput`, `PageRequest`, etc. |
| `internal/backend/appcontext/context.go` | **Modify** | Add `CarrierService *services.CarrierService` + initialisation |
| `internal/backend/handlers/graphql/router.go` | **Modify** | Wire gqlgen handler with `NewResolver(appCtx.CarrierService)` |

---

## 12. Generation Commands

Run these in order after making the schema and SQL changes:

```bash
# 1. Regenerate sqlc database layer (after appending to carriers.sql)
sqlc generate

# 2. Verify the whole repository still compiles
go build ./...

# 3. Regenerate gqlgen GraphQL layer (after updating .graphql files and gqlgen.yml)
go run github.com/99designs/gqlgen generate

# 4. Tidy dependencies if gqlgen pulled anything new
go mod tidy

# 5. Verify the full build again and run all tests
go build ./...
go test ./...
```

> **Important:** gqlgen's `generate` command will regenerate the resolver skeleton files.
> It preserves existing method bodies but will add new method stubs. Always review the
> diff after running `gqlgen generate` to confirm nothing was overwritten unexpectedly.
