# BACKEND_GRAPHQL.md

GraphQL API design and implementation guide for the `charging-backend` application.

This document covers the common pagination/filter framework shared by all list endpoints,
the first concrete resource implementation (**CarrierResource**), and the second:
**ClassificationResource**.

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
13. [Pre-work: Model Package Refactor](#13-pre-work-model-package-refactor)
14. [ClassificationResource — Design](#14-classificationresource--design)
15. [ClassificationResource — GraphQL Schema](#15-classificationresource--graphql-schema)
16. [ClassificationResource — SQL Queries](#16-classificationresource--sql-queries)
17. [ClassificationResource — Store Layer](#17-classificationresource--store-layer)
18. [ClassificationResource — Service Layer](#18-classificationresource--service-layer)
19. [ClassificationResource — Resolver Layer](#19-classificationresource--resolver-layer)
20. [ClassificationResource — File Map](#20-classificationresource--file-map)

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

---

## 13. Pre-work: Model Package Refactor

Before implementing the ClassificationResource two preparatory changes must be made.
Both are **pure refactors** — no behaviour changes, no new files outside the model package.

### 13.1 Rename `Plan` → `ClassificationPlan`

`internal/chargeengine/model/classificationplan.go` currently names the root struct `Plan`.
This is ambiguous (every resource has a "plan") and conflicts with the new GraphQL type name.
Rename it to `ClassificationPlan` throughout.

Files affected by the rename (use your editor's global rename):

| File | Change |
|---|---|
| `internal/chargeengine/model/classificationplan.go` | `type Plan struct` → `type ClassificationPlan struct` |
| `internal/chargeengine/model/classificationplan_test.go` | All `model.Plan{...}` → `model.ClassificationPlan{...}` |
| `internal/chargeengine/engine/providers/classificationplan/classificationprovider.go` | `*model.Plan` → `*model.ClassificationPlan` (×4) |
| `internal/chargeengine/engine/providers/classificationplan/classificationprovider_test.go` | Test helper types if any |
| `internal/chargeengine/engine/business/interfaces/infra.go` | `FetchClassificationPlan() (*model.Plan, error)` |
| `internal/chargeengine/engine/steps/classification-step_test.go` | Mock return type `*model.Plan` |

> **Note:** `classificationprovider.go` also uses `model.Plan{}` as a local variable — update all
> instances, not just the function signatures.

### 13.2 Move `internal/chargeengine/model/` → `internal/model/`

The model package is now required by both `charging-engine` and `charging-backend`. Keeping it
under `internal/chargeengine/` signals it is private to that application, which is no longer true.

**Procedure:**

```bash
# 1. Create the new package directory
mkdir -p internal/model

# 2. Copy all non-test source files (test files move too)
#    Then update the package declaration from "package model" (unchanged — same name).
#    Only the import path changes.

# 3. Global search/replace in the repository:
#    OLD:  "go-ocs/internal/chargeengine/model"
#    NEW:  "go-ocs/internal/model"
#
#    25 files require this update (confirmed by grep):
```

Files that import `go-ocs/internal/chargeengine/model` (25 files):

```
internal/chargeengine/engine/steps/classification-step_test.go
internal/chargeengine/engine/steps/trace-step_test.go
internal/chargeengine/engine/steps/chargedata-step_test.go
internal/chargeengine/engine/steps/error-step_test.go
internal/chargeengine/engine/steps/response-step_test.go
internal/chargeengine/engine/steps/accounting-step_test.go
internal/chargeengine/engine/steps/rating-step_test.go
internal/chargeengine/chargeservice_test.go
internal/chargeengine/engine/steps/authentication-step_test.go
internal/chargeengine/engine/business/classifying.go
internal/chargeengine/engine/steps/accounting-step.go
internal/chargeengine/engine/steps/rating-step.go
internal/chargeengine/engine/chargingcontext.go
internal/chargeengine/engine/business/rating_test.go
internal/chargeengine/engine/business/classifying_test.go
internal/chargeengine/engine/business/rating.go
internal/chargeengine/engine/servicecontext.go
internal/chargeengine/engine/business/interfaces/infra.go
internal/chargeengine/engine/providers/subscribers/subscriberprovider_test.go
internal/chargeengine/engine/providers/subscribers/subscriberprovider.go
internal/chargeengine/engine/providers/ratingplan/ratingplanprovider_test.go
internal/chargeengine/engine/providers/classificationplan/classificationprovider.go
internal/chargeengine/engine/providers/classificationplan/classificationprovider_test.go
internal/chargeengine/engine/providers/ratingplan/ratingplanprovider.go
internal/chargeengine/engine/steps/chargedata-step.go
```

```bash
# 4. Delete the old package directory after confirming the build passes
rm -rf internal/chargeengine/model

# 5. Verify
go build ./...
go test ./...
```

> **Commit separately:** The rename + move should be a single dedicated commit
> (`refactor: move model package to internal/model and rename Plan to ClassificationPlan`)
> before any ClassificationResource code is added. This keeps the diff clean and reviewable.

---

## 14. ClassificationResource — Design

### 14.1 Domain Overview

The `classification` table stores versioned Classification Plans. Each row is a full
classification configuration (the embedded `plan` JSONB column) plus lifecycle metadata.

**Table schema (from `000001_init.up.sql`):**

| Column | Type | Notes |
|---|---|---|
| `classification_id` | `uuid` PK | Generated by the application (not the DB) |
| `name` | `varchar` NOT NULL | Human-readable label |
| `created_on` | `timestamp` DEFAULT now() | Set by DB |
| `effective_time` | `TIMESTAMPTZ` NOT NULL | When the plan becomes effective |
| `created_by` | `varchar` NOT NULL | Username from JWT |
| `approved_by` | `varchar` nullable | Username from JWT at approval |
| `status` | `varchar` DEFAULT `'DRAFT'` | State machine: DRAFT → PENDING → ACTIVE / DRAFT |
| `plan` | `jsonb` NOT NULL | Serialised `ClassificationPlan` struct |

**Status state machine (mirrors Java `Status` enum):**

```
         submitForApproval          approve
  DRAFT ─────────────────► PENDING ────────► ACTIVE
    ▲                          │
    └──────── decline ─────────┘
              (status resets to DRAFT)

  Any DRAFT may be deleted.
  Active and RETIRED plans are read-only.
```

### 14.2 The `rateKeyInput()` Query

This query powers the frontend dropdowns used when building rate plans. It derives its data
from the **active** classification plan (the one currently driving charging decisions).

The service:
1. Calls `FindActiveClassification` to get the raw JSONB.
2. Unmarshals into `model.ClassificationPlan`.
3. Iterates `ServiceTypes` to extract unique values for each lookup list.

| Response field | Source |
|---|---|
| `serviceTypes` | Unique `st.ServiceType` values across all `ServiceTypes` |
| `sourceTypes` | Unique `st.SourceType` values |
| `serviceDirections` | Unique `st.ServiceDirection` values (typically `MO`, `MT`) |
| `serviceCategories` | All entries in `st.ServiceCategoryMap` (key → code, value → name), with `serviceTypeCode = st.ServiceType` |
| `serviceWindows` | All entries in `st.ServiceWindows` (the per-service-type slice of window names), with `serviceTypeCode = st.ServiceType` |

### 14.3 Map Flattening

The `ClassificationPlan` struct uses Go maps (`map[string]ServiceWindow`,
`map[string]string`) for two fields. GraphQL has no native map type, so both are
represented as key-value lists:

| Go field | GraphQL type |
|---|---|
| `ClassificationPlan.ServiceWindows map[string]ServiceWindow` | `[ServiceWindowEntry!]` where `ServiceWindowEntry { name, startTime, endTime }` |
| `ServiceType.ServiceCategoryMap map[string]string` | `[ServiceCategoryMapEntry!]` where `ServiceCategoryMapEntry { key, value }` |

The service layer handles the conversion in both directions:
- **Read path:** Go map → GraphQL list (iterate map, emit one entry per key).
- **Write path:** GraphQL list → Go map (build map from input entries).

### 14.4 `createdBy` / `approvedBy` Population

`createdBy` (on create/clone) and `approvedBy` (on approve/decline) are populated from
the authenticated user's JWT, not from the client payload. The value used is
`KeycloakClaims.Email` — the verified email address of the authenticated user.

### 14.5 `cloneClassification` Behaviour

The clone:
- Gets a new `classification_id` (new UUID).
- Inherits `name`, `effectiveTime`, and `plan` from the source.
- Status is forced to `DRAFT`.
- `createdBy` is the current user (from JWT). `approvedBy` is cleared.
- `created_on` is set to now() by the database.

---

## 15. ClassificationResource — GraphQL Schema

**File:** `gql/schema/classification.graphql` — **new file**

```graphql
# Classification status lifecycle.
# Transitions: DRAFT → PENDING → ACTIVE (approve) or back to DRAFT (decline).
# Only DRAFT plans may be edited or deleted.
enum ClassificationStatus {
  DRAFT
  PENDING
  ACTIVE
  RETIRED
}

# The full classification entity — metadata wrapper around an embedded ClassificationPlan.
# Mirrors Java ClassificationEntity.
type Classification {
  classificationId: ID!
  name:             String!
  createdOn:        DateTime
  effectiveTime:    DateTime!
  createdBy:        String!
  approvedBy:       String
  status:           ClassificationStatus!
  plan:             ClassificationPlan!
}

# The embedded classification plan (stored as JSONB).
# Mirrors the Go model.ClassificationPlan struct (formerly Plan).
type ClassificationPlan {
  ruleSetId:            String
  ruleSetName:          String
  useServiceWindows:    Boolean!
  defaultServiceWindow: String!
  defaultSourceType:    String!
  serviceWindows:       [ServiceWindowEntry!]
  serviceTypes:         [ClassificationServiceType!]
}

# A named service window (flattened from map[string]ServiceWindow).
type ServiceWindowEntry {
  name:      String!
  startTime: String!   # "HH:mm" format
  endTime:   String!   # "HH:mm" format
}

# A single service type classification rule.
type ClassificationServiceType {
  type:                   String!
  chargingInformation:    String!
  serviceTypeRule:        String
  description:            String
  sourceType:             String!
  serviceDirection:       String!
  serviceCategory:        String!
  serviceIdentifier:      String
  defaultServiceCategory: String
  unitType:               String!
  serviceWindows:         [String!]
  serviceCategoryMap:     [ServiceCategoryMapEntry!]
}

# A key-value pair (flattened from map[string]string).
type ServiceCategoryMapEntry {
  key:   String!
  value: String!
}

# ---------------------------------------------------------------------------
# RateKeyInput — lookup data derived from the active classification plan.
# Used by the frontend to populate dropdowns when building rate plans.
# Mirrors Java RateKeyInputResponseDto.
# ---------------------------------------------------------------------------

type LookupData {
  code: String!
  name: String!
}

type ServiceCategoryLookup {
  code:            String!
  name:            String!
  serviceTypeCode: String!
}

type RateKeyInput {
  serviceTypes:      [LookupData!]!
  sourceTypes:       [LookupData!]!
  serviceDirections: [LookupData!]!
  serviceCategories: [ServiceCategoryLookup!]!
  serviceWindows:    [ServiceCategoryLookup!]!
}

# ---------------------------------------------------------------------------
# Input types
# ---------------------------------------------------------------------------

input ClassificationInput {
  name:          String!
  effectiveTime: DateTime!
  plan:          ClassificationPlanInput!
}

input ClassificationPlanInput {
  ruleSetId:            String
  ruleSetName:          String
  useServiceWindows:    Boolean!
  defaultServiceWindow: String!
  defaultSourceType:    String!
  serviceWindows:       [ServiceWindowEntryInput!]
  serviceTypes:         [ClassificationServiceTypeInput!]!
}

input ServiceWindowEntryInput {
  name:      String!
  startTime: String!
  endTime:   String!
}

input ClassificationServiceTypeInput {
  type:                   String!
  chargingInformation:    String!
  serviceTypeRule:        String
  description:            String
  sourceType:             String!
  serviceDirection:       String!
  serviceCategory:        String!
  serviceIdentifier:      String
  defaultServiceCategory: String
  unitType:               String!
  serviceWindows:         [String!]
  serviceCategoryMap:     [ServiceCategoryMapEntryInput!]
}

input ServiceCategoryMapEntryInput {
  key:   String!
  value: String!
}

# ---------------------------------------------------------------------------
# Queries
# ---------------------------------------------------------------------------

extend type Query {
  # Returns a filtered, sorted, paginated list of classification plans.
  classificationList(page: PageRequest, filter: FilterRequest): [Classification!]!

  # Returns the total count of classification plans matching the filter.
  countClassifications(filter: FilterRequest): Int!

  # Returns lookup data derived from the active classification plan.
  # Used to populate rate-plan configuration dropdowns in the frontend.
  rateKeyInput: RateKeyInput!

  # Returns a single classification plan by ID, or null if not found.
  classification(classificationId: ID!): Classification
}

# ---------------------------------------------------------------------------
# Mutations
# ---------------------------------------------------------------------------

extend type Mutation {
  # Creates a new classification plan in DRAFT status.
  # createdBy is derived from the authenticated JWT; it is not a client input.
  createClassification(classification: ClassificationInput!): Classification!

  # Creates a DRAFT copy of an existing classification plan.
  cloneClassification(classificationId: ID!): Classification!

  # Updates the name, effectiveTime, and plan of a DRAFT classification.
  # Returns an error if the classification is not in DRAFT status.
  updateClassificationPlan(classificationId: ID!, classification: ClassificationInput!): Classification!

  # Transitions a DRAFT classification to PENDING (awaiting approval).
  submitClassificationForApproval(classificationId: ID!): Classification!

  # Transitions a PENDING classification to ACTIVE.
  # approvedBy is derived from the authenticated JWT.
  approveClassificationPlan(classificationId: ID!): Classification!

  # Transitions a PENDING classification back to DRAFT.
  declineClassificationPlan(classificationId: ID!): Classification!

  # Permanently deletes a DRAFT classification. Returns true on success.
  # Returns an error if the classification is not in DRAFT status.
  deleteClassification(classificationId: ID!): Boolean!
}
```

---

## 16. ClassificationResource — SQL Queries

**File:** `db/queries/classification.sql` — new file (or append to existing)

The existing `FindActiveClassification` query is retained in `classification.sql.go` (generated).
All new queries are added to this file then `sqlc generate` is run.

```sql
-- name: FindClassificationByID :one
-- Retrieves a single classification record by its UUID.
SELECT classification_id, name, created_on, effective_time,
       created_by, approved_by, status, plan
FROM classification
WHERE classification_id = $1;

-- name: CreateClassification :one
-- Inserts a new classification in DRAFT status and returns the full persisted row.
-- classification_id is generated by the application (uuid.New()).
INSERT INTO classification (classification_id, name, effective_time, created_by, plan, status)
VALUES ($1, $2, $3, $4, $5, 'DRAFT')
RETURNING classification_id, name, created_on, effective_time,
          created_by, approved_by, status, plan;

-- name: UpdateClassificationPlan :one
-- Updates the name, effectiveTime, and plan of an existing DRAFT classification.
-- Returns an error if no row matches (classification not found or not DRAFT).
UPDATE classification
SET name           = $2,
    effective_time = $3,
    plan           = $4
WHERE classification_id = $1
  AND status = 'DRAFT'
RETURNING classification_id, name, created_on, effective_time,
          created_by, approved_by, status, plan;

-- name: SubmitClassification :one
-- Transitions a DRAFT classification to PENDING (submitted for approval).
UPDATE classification
SET status = 'PENDING'
WHERE classification_id = $1
  AND status = 'DRAFT'
RETURNING classification_id, name, created_on, effective_time,
          created_by, approved_by, status, plan;

-- name: ApproveClassification :one
-- Transitions a PENDING classification to ACTIVE and records the approver.
UPDATE classification
SET status      = 'ACTIVE',
    approved_by = $2
WHERE classification_id = $1
  AND status = 'PENDING'
RETURNING classification_id, name, created_on, effective_time,
          created_by, approved_by, status, plan;

-- name: DeclineClassification :one
-- Transitions a PENDING classification back to DRAFT, clearing the approver.
UPDATE classification
SET status      = 'DRAFT',
    approved_by = NULL
WHERE classification_id = $1
  AND status = 'PENDING'
RETURNING classification_id, name, created_on, effective_time,
          created_by, approved_by, status, plan;

-- name: DeleteClassification :exec
-- Permanently deletes a DRAFT classification.
-- The application layer must verify status = DRAFT before calling this.
DELETE FROM classification
WHERE classification_id = $1
  AND status = 'DRAFT';
```

> **Note on clone:** `cloneClassification` is implemented at the application layer by calling
> `FindClassificationByID` then `CreateClassification` with a new UUID — no dedicated SQL query.

> **Note on dynamic queries:** `classificationList` and `countClassifications` use runtime-constructed
> WHERE clauses (same pattern as carrier). These are in the store layer (Section 17), not sqlc.

---

## 17. ClassificationResource — Store Layer

**File:** `internal/store/classification_store.go` — new hand-written file.

Mirrors `carrier_store.go`. Wildcard columns match Java `ClassificationEntity.WILDCARD_FIELDS`:
`classificationId`, `name`, `status`.

```go
package store

import (
    "context"
    "fmt"

    "go-ocs/internal/store/sqlc"
)

// ListClassificationsParams holds runtime-constructed SQL fragments for a dynamic
// classification query. WhereSQL and Args come from filter.BuildWhere;
// OrderSQL from filter.BuildOrderBy.
type ListClassificationsParams struct {
    WhereSQL string
    Args     []any
    OrderSQL string
    Limit    int
    Offset   int
}

// ListClassifications executes a dynamic classification query with optional filtering,
// sorting, and pagination.
func (s *Store) ListClassifications(
    ctx context.Context,
    p ListClassificationsParams,
) ([]sqlc.Classification, error) {
    limitIdx  := len(p.Args) + 1
    offsetIdx := limitIdx + 1

    q := fmt.Sprintf(
        `SELECT classification_id, name, created_on, effective_time,
                created_by, approved_by, status, plan
         FROM classification %s %s LIMIT $%d OFFSET $%d`,
        p.WhereSQL, p.OrderSQL, limitIdx, offsetIdx,
    )
    args := append(p.Args, p.Limit, p.Offset)

    rows, err := s.DB.Query(ctx, q, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var items []sqlc.Classification
    for rows.Next() {
        var c sqlc.Classification
        if err := rows.Scan(
            &c.ClassificationID, &c.Name, &c.CreatedOn, &c.EffectiveTime,
            &c.CreatedBy, &c.ApprovedBy, &c.Status, &c.Plan,
        ); err != nil {
            return nil, err
        }
        items = append(items, c)
    }
    return items, rows.Err()
}

// CountClassifications executes a dynamic count query with optional filtering.
func (s *Store) CountClassifications(
    ctx context.Context,
    whereSQL string,
    args []any,
) (int64, error) {
    q := fmt.Sprintf("SELECT COUNT(*) FROM classification %s", whereSQL)
    var n int64
    if err := s.DB.QueryRow(ctx, q, args...).Scan(&n); err != nil {
        return 0, err
    }
    return n, nil
}
```

---

## 18. ClassificationResource — Service Layer

**File:** `internal/backend/services/classification_service.go` — new file.

### 18.1 Column / Wildcard Maps

```go
// classificationColumns maps GraphQL field names to SQL column names.
var classificationColumns = map[string]string{
    "classificationId": "classification_id",
    "name":             "name",
    "status":           "status",
    "createdBy":        "created_by",
    "approvedBy":       "approved_by",
    "effectiveTime":    "effective_time",
    "createdOn":        "created_on",
}

// classificationWildcardCols mirrors Java ClassificationEntity.WILDCARD_FIELDS.
var classificationWildcardCols = []string{
    "classification_id", "name", "status",
}
```

### 18.2 Service Methods

```go
type ClassificationService struct {
    store *store.Store
}

func NewClassificationService(s *store.Store) *ClassificationService

// ListClassifications — paginated, filtered list.
func (s *ClassificationService) ListClassifications(
    ctx context.Context,
    page *graphqlmodel.PageRequest,
    filterReq *graphqlmodel.FilterRequest,
) ([]*graphqlmodel.Classification, error)

// CountClassifications — total count with optional filter.
func (s *ClassificationService) CountClassifications(
    ctx context.Context,
    filterReq *graphqlmodel.FilterRequest,
) (int, error)

// GetClassification — single lookup by ID.
func (s *ClassificationService) GetClassification(
    ctx context.Context,
    id string,
) (*graphqlmodel.Classification, error)

// RateKeyInput — derives lookup data from the active classification plan.
func (s *ClassificationService) RateKeyInput(
    ctx context.Context,
) (*graphqlmodel.RateKeyInput, error)

// CreateClassification — inserts a new DRAFT.
// createdBy is extracted from ctx (JWT claims).
func (s *ClassificationService) CreateClassification(
    ctx context.Context,
    input graphqlmodel.ClassificationInput,
) (*graphqlmodel.Classification, error)

// CloneClassification — creates a DRAFT copy of an existing classification.
// createdBy is extracted from ctx (JWT claims).
func (s *ClassificationService) CloneClassification(
    ctx context.Context,
    classificationId string,
) (*graphqlmodel.Classification, error)

// UpdateClassificationPlan — updates name, effectiveTime, plan of a DRAFT.
func (s *ClassificationService) UpdateClassificationPlan(
    ctx context.Context,
    classificationId string,
    input graphqlmodel.ClassificationInput,
) (*graphqlmodel.Classification, error)

// SubmitClassificationForApproval — DRAFT → PENDING.
func (s *ClassificationService) SubmitClassificationForApproval(
    ctx context.Context,
    classificationId string,
) (*graphqlmodel.Classification, error)

// ApproveClassificationPlan — PENDING → ACTIVE.
// approvedBy is extracted from ctx (JWT claims).
func (s *ClassificationService) ApproveClassificationPlan(
    ctx context.Context,
    classificationId string,
) (*graphqlmodel.Classification, error)

// DeclineClassificationPlan — PENDING → DRAFT.
func (s *ClassificationService) DeclineClassificationPlan(
    ctx context.Context,
    classificationId string,
) (*graphqlmodel.Classification, error)

// DeleteClassification — removes a DRAFT. Returns true on success.
func (s *ClassificationService) DeleteClassification(
    ctx context.Context,
    classificationId string,
) (bool, error)
```

### 18.3 Model Mapping

The service is responsible for translating between:
- `sqlc.Classification` (raw DB row with JSONB `Plan []byte`)
- `model.ClassificationPlan` (the decoded Go struct, formerly `Plan`)
- `graphqlmodel.Classification` and `graphqlmodel.ClassificationPlan` (the gqlgen-generated types)

**Key mapping decisions:**

| Mapping | Detail |
|---|---|
| `sqlc.Classification.Plan []byte` → `model.ClassificationPlan` | `json.Unmarshal` |
| `model.ClassificationPlan` → `graphqlmodel.ClassificationPlan` | Field-by-field; maps flattened to slices |
| `model.ClassificationPlan.ServiceWindows map[string]ServiceWindow` → `[]*graphqlmodel.ServiceWindowEntry` | Iterate map; emit `{Name: k, StartTime: sw.StartTime.Format("15:04"), EndTime: sw.EndTime.Format("15:04")}` |
| `graphqlmodel.ServiceWindowEntryInput` → `map[string]model.ServiceWindow` | Build map from input list; parse `"HH:mm"` strings back to `common.LocalTime` |
| `model.ServiceType.ServiceCategoryMap map[string]string` → `[]*graphqlmodel.ServiceCategoryMapEntry` | Iterate map; emit `{Key: k, Value: v}` |
| `graphqlmodel.ServiceCategoryMapEntryInput` → `map[string]string` | Build map from input list |
| `sqlc.Classification.EffectiveTime pgtype.Timestamptz` → `*string` (DateTime scalar) | Format as RFC3339 |
| `graphqlmodel.ClassificationInput.EffectiveTime string` → `pgtype.Timestamptz` | Parse RFC3339 |

### 18.4 `RateKeyInput` Derivation Logic

```
1. rec = store.Q.FindActiveClassification(ctx)       // returns sqlc.Classification
2. plan = json.Unmarshal(rec.Plan, &model.ClassificationPlan{})
3. For each st in plan.ServiceTypes:
   a. serviceTypes:      add {code: st.ServiceType, name: st.ServiceType} if not seen
   b. sourceTypes:       add {code: st.SourceType, name: st.SourceType} if not seen
   c. serviceDirections: add {code: st.ServiceDirection, name: st.ServiceDirection} if not seen
   d. serviceCategories: for each (k, v) in st.ServiceCategoryMap:
                           add {code: k, name: v, serviceTypeCode: st.ServiceType}
   e. serviceWindows:    for each windowName in st.ServiceWindows:
                           add {code: windowName, name: windowName, serviceTypeCode: st.ServiceType}
4. Return RateKeyInput{...}
```

### 18.5 `createdBy` / `approvedBy` Extraction

```go
// emailFromContext extracts the authenticated user's email address from the request context.
// Uses keycloak.ClaimsFromContext (defined in internal/auth/keycloak/middleware.go).
// Returns "unknown" if auth is disabled or no claims are present.
func emailFromContext(ctx context.Context) string {
    claims, ok := keycloak.ClaimsFromContext(ctx)
    if !ok || claims == nil {
        return "unknown"
    }
    return claims.Email
}
```

> `keycloak.ClaimsFromContext` reads from `ClaimsContextKey` ("keycloak_claims") which the
> auth middleware stores on every authenticated request. When auth is disabled the middleware
> is a no-op, so no claims are present and "unknown" is returned.

---

## 19. ClassificationResource — Resolver Layer

**File:** `internal/backend/resolvers/classification.resolvers.go` (gqlgen-generated skeleton, bodies filled in)

All methods delegate to `ClassificationService` — zero business logic in the resolver.

```go
// Queries
func (r *queryResolver) ClassificationList(ctx, page, filter) ([]*model.Classification, error)
    → r.ClassificationSvc.ListClassifications(ctx, page, filter)

func (r *queryResolver) CountClassifications(ctx, filter) (int, error)
    → r.ClassificationSvc.CountClassifications(ctx, filter)

func (r *queryResolver) RateKeyInput(ctx) (*model.RateKeyInput, error)
    → r.ClassificationSvc.RateKeyInput(ctx)

func (r *queryResolver) Classification(ctx, classificationId) (*model.Classification, error)
    → r.ClassificationSvc.GetClassification(ctx, classificationId)

// Mutations
func (r *mutationResolver) CreateClassification(ctx, classification) (*model.Classification, error)
    → r.ClassificationSvc.CreateClassification(ctx, classification)

func (r *mutationResolver) CloneClassification(ctx, classificationId) (*model.Classification, error)
    → r.ClassificationSvc.CloneClassification(ctx, classificationId)

func (r *mutationResolver) UpdateClassificationPlan(ctx, classificationId, classification) (*model.Classification, error)
    → r.ClassificationSvc.UpdateClassificationPlan(ctx, classificationId, classification)

func (r *mutationResolver) SubmitClassificationForApproval(ctx, classificationId) (*model.Classification, error)
    → r.ClassificationSvc.SubmitClassificationForApproval(ctx, classificationId)

func (r *mutationResolver) ApproveClassificationPlan(ctx, classificationId) (*model.Classification, error)
    → r.ClassificationSvc.ApproveClassificationPlan(ctx, classificationId)

func (r *mutationResolver) DeclineClassificationPlan(ctx, classificationId) (*model.Classification, error)
    → r.ClassificationSvc.DeclineClassificationPlan(ctx, classificationId)

func (r *mutationResolver) DeleteClassification(ctx, classificationId) (bool, error)
    → r.ClassificationSvc.DeleteClassification(ctx, classificationId)
```

**`resolver.go` additions:**

```go
type Resolver struct {
    CarrierSvc        *services.CarrierService
    ClassificationSvc *services.ClassificationService   // ADD
}
```

---

## 20. ClassificationResource — File Map

| File | Action | Notes |
|---|---|---|
| `internal/chargeengine/model/classificationplan.go` | **Modify** | Rename `Plan` → `ClassificationPlan` |
| *(25 files)* | **Modify** | Update import path from `chargeengine/model` → `model` |
| `gql/schema/classification.graphql` | **New** | Full schema: types, inputs, queries, mutations |
| `db/queries/classification.sql` | **New** | 7 new sqlc queries (FindByID, Create, Update, Submit, Approve, Decline, Delete) |
| `internal/store/sqlc/classification.sql.go` | **Regenerate** | Run `sqlc generate` |
| `internal/store/classification_store.go` | **New** | `ListClassifications()`, `CountClassifications()` dynamic pgx methods |
| `internal/backend/services/classification_service.go` | **New** | Full `ClassificationService` implementation |
| `internal/backend/resolvers/classification.resolvers.go` | **New** | gqlgen-generated skeleton + resolver bodies |
| `internal/backend/resolvers/resolver.go` | **Modify** | Add `ClassificationSvc *services.ClassificationService` |
| `internal/backend/appcontext/context.go` | **Modify** | Add `ClassificationSvc` field + initialisation |
| `internal/backend/handlers/graphql/router.go` | **Modify** | Pass `ClassificationSvc` to `resolvers.NewResolver(...)` |
| `internal/backend/graphql/generated/generated.go` | **Regenerate** | Run `gqlgen generate` |
| `internal/backend/graphql/model/models_gen.go` | **Regenerate** | Run `gqlgen generate` |
| `internal/backend/services/classification_service_test.go` | **New** | Unit tests following carrier_service_test.go pattern |

---

## 21. Pre-work: Multi-Tenant Middleware

The RatePlan resource requires a **tenant resolver** that maps the HTTP `Host` header to the
wholesaler's UUID. This UUID is stored in the request context and consumed by the RETAIL rate
plan operations (`createRatePlan`, `latestRatePlanList`).

### 21.1 Design

| Concern | Decision |
|---|---|
| Storage | In-memory `map[string]pgtype.UUID` (hostname → wholesaler UUID) |
| Data source | `wholesaler.hosts` column (PostgreSQL text array) |
| Refresh | Background goroutine on configurable interval (default 10 minutes) |
| Context key | `"tenant_wholesale_id"` stored by middleware, read via `tenant.WholesaleIDFromContext(ctx)` |
| No match | Request proceeds without a wholesale ID in context; service layer decides whether to error |

### 21.2 Configuration addition

```yaml
# cmd/charging-backend/backend-config.yaml
server:
  tenantRefreshInterval: 10m   # how often to reload the hostname→wholesaler map
```

```go
// internal/backend/appcontext/config.go
type ServerConfig struct {
    // ... existing fields ...
    TenantRefreshInterval time.Duration `yaml:"tenantRefreshInterval"`
}
```

### 21.3 New package: `internal/auth/tenant/`

**`resolver.go`** — loads wholesalers, builds map, runs background refresh:
```go
type Resolver struct {
    store    *store.Store
    mu       sync.RWMutex
    hostsMap map[string]pgtype.UUID   // hostname → wholesaler UUID
    interval time.Duration
}

func NewResolver(s *store.Store, interval time.Duration) *Resolver
func (r *Resolver) Start(ctx context.Context)                      // background refresh goroutine
func (r *Resolver) refresh(ctx context.Context)                    // load from DB, rebuild map
func (r *Resolver) resolveHost(host string) (pgtype.UUID, bool)   // strip port, lookup
```

**`middleware.go`** — HTTP middleware that resolves Host → wholesaleId → context:
```go
const WholesaleIDContextKey contextKey = "tenant_wholesale_id"

func Middleware(r *Resolver) func(http.Handler) http.Handler
func WholesaleIDFromContext(ctx context.Context) (pgtype.UUID, bool)
```

### 21.4 SQL required

New query in `internal/store/queries/wholesaler.sql`:
```sql
-- name: AllWholesalers :many
-- Returns all wholesaler records for building the hostname lookup table.
SELECT id, hosts FROM wholesaler WHERE active = true;
```

### 21.5 Wiring

- `appcontext.AppContext` gains `TenantResolver *tenant.Resolver`
- `NewAppContext` calls `tenant.NewResolver(s, cfg.Server.TenantRefreshInterval)`
- `main.go` calls `appCtx.TenantResolver.Start(ctx)` after context is built
- `router.go` inserts `tenant.Middleware(appCtx.TenantResolver)` before the keycloak middleware

---

## 22. RatePlanResource — Design

### 22.1 Domain Overview

The `rateplan` table stores versioned rate plans. A **logical plan** is identified by `plan_id`
(UUID). Multiple rows may share the same `plan_id` — each row is a separate version with its own
`effective_at` timestamp and lifecycle status.

**The "current effective" plan** for charging is determined by the charging engine as:
`WHERE plan_status = 'ACTIVE' AND effective_at <= NOW() ORDER BY effective_at DESC LIMIT 1`.

**Table columns:**

| Column | Type | Notes |
|---|---|---|
| `id` | `bigserial` PK | Internal auto-increment — **not exposed in GraphQL** |
| `plan_id` | `uuid` NOT NULL | Logical plan identifier; groups versions |
| `modified_at` | `TIMESTAMPTZ` | Managed by the DB / application |
| `plan_type` | `varchar` | `SETTLEMENT`, `WHOLESALE`, `RETAIL` |
| `wholesale_id` | `uuid` nullable | Null for SETTLEMENT/WHOLESALE; set for RETAIL |
| `plan_name` | `varchar` | Human-readable label |
| `rateplan` | `jsonb` | Serialised `model.RatePlan` struct |
| `plan_status` | `varchar` DEFAULT `'DRAFT'` | State machine |
| `created_by` | `varchar` | Email from JWT |
| `approved_by` | `varchar` nullable | Email from JWT at approval |
| `effective_at` | `TIMESTAMPTZ` | When the plan version takes effect |

### 22.2 Status State Machine

```
         submitForApproval            approve
  DRAFT ───────────────────► PENDING ────────► ACTIVE ──► (RETIRED — future)
    ▲                            │
    └────────── decline ─────────┘
                (resets to DRAFT)

  Rules:
  - Only DRAFT versions may be edited (updateRatePlan, updateRatePlanRules)
  - Only DRAFT versions may be deleted
  - ACTIVE versions are IMMUTABLE — no exceptions
  - Multiple ACTIVE versions of the same plan_id may exist; effective_at determines precedence
```

### 22.3 wholesaleId Handling

| Plan type | wholesaleId |
|---|---|
| `SETTLEMENT` | Always null |
| `WHOLESALE` | Always null |
| `RETAIL` | Set from the tenant middleware context (resolved from Host header) |

`createRatePlan`: if `planType == RETAIL`, the service reads `tenant.WholesaleIDFromContext(ctx)`.
If not present, returns an error. For other types, sets wholesaleId to null.

### 22.4 Versioning and Clone

- `createRatePlan` always generates a **new** `plan_id` UUID — this is a brand new logical plan.
- `cloneRatePlan(planId)` finds the **latest version** of the source `plan_id` (highest
  `effective_at, id DESC`) and creates a new DRAFT row with the **same** `plan_id`. This produces
  a new version of the same logical plan, ready for editing while the current version remains active.
  `createdBy` is the current JWT user; `approvedBy` is cleared; `plan_status` is `DRAFT`.

### 22.5 `latestRatePlanList`

Returns the **most recent version** (by `effective_at DESC, id DESC`) of each `plan_id` for a
given `planType`. For `RETAIL`, also scopes to `wholesale_id` from the tenant context.

This query is only called from a RETAIL portal context. If called for `RETAIL` and no
`wholesale_id` is in context, the service returns an error.

### 22.6 JSONB Sync

The `rateplan` JSONB column stores a `model.RatePlan` struct, which includes embedded
`RatePlanID`, `RatePlanName`, `RatePlanType`, and `EffectiveFrom` fields that mirror the table
columns. The service keeps these in sync on every create/update so the charging engine always
reads a consistent embedded document.

### 22.7 updateRatePlan vs updateRatePlanRules

Both mutations only operate on DRAFT plans. For now both are implemented:
- `updateRatePlan` — updates `plan_name`, `plan_type`, `effective_at`, and the full `rateplan` JSONB.
- `updateRatePlanRules` — updates only the `rateplan` JSONB (rate lines), leaving metadata unchanged.

> **TODO:** Verify with the Java source which mutation is actually used in production and remove
> the unused one.

---

## 23. RatePlanResource — GraphQL Schema

**File:** `gql/schema/rateplan.graphql` — new file

```graphql
enum RatePlanType {
  SETTLEMENT
  WHOLESALE
  RETAIL
}

enum RatePlanStatus {
  DRAFT
  PENDING
  ACTIVE
  RETIRED
}

# A versioned rate plan. planId groups logical versions; effective_at determines precedence.
# The internal bigserial id column is not exposed — planId is the GraphQL identifier.
type RatePlan {
  planId:      ID!
  planName:    String!
  planType:    RatePlanType!
  wholesaleId: String           # null for SETTLEMENT and WHOLESALE plans
  planStatus:  RatePlanStatus!
  createdBy:   String!
  approvedBy:  String
  effectiveAt: DateTime!
  modifiedAt:  DateTime
  rateLines:   [RateLine!]!
}

# A single rate line within a rate plan.
# classificationKey is a dot-separated string: serviceType.sourceType.direction.category[.window]
# Use the rateKeyInput query to obtain valid values for each component.
# baseTariff and multiplier are decimal strings to preserve monetary precision.
type RateLine {
  classificationKey: String!
  groupKey:          String
  description:       String
  tariffType:        String!     # ACTUAL | PERCENTAGE | MARKUP
  unitType:          String!     # SECONDS | OCTETS | UNITS | MONETARY
  baseTariff:        String!
  unitOfMeasure:     Int!
  multiplier:        String!
  qosProfile:        String
  minimumUnits:      Int!
  roundingIncrement: Int!
  barred:            Boolean!
  monetaryOnly:      Boolean!
}

# ---------------------------------------------------------------------------
# Input types
# ---------------------------------------------------------------------------

input RatePlanInput {
  planName:    String!
  planType:    RatePlanType!
  effectiveAt: DateTime!
  rateLines:   [RateLineInput!]!
}

input RateLineInput {
  classificationKey: String!
  groupKey:          String
  description:       String
  tariffType:        String!
  unitType:          String!
  baseTariff:        String!
  unitOfMeasure:     Int!
  multiplier:        String!
  qosProfile:        String
  minimumUnits:      Int!
  roundingIncrement: Int!
  barred:            Boolean!
  monetaryOnly:      Boolean!
}

# ---------------------------------------------------------------------------
# Queries
# ---------------------------------------------------------------------------

extend type Query {
  # Returns a filtered, sorted, paginated list of rate plans (all versions).
  ratePlanList(page: PageRequest, filter: FilterRequest): [RatePlan!]!

  # Returns the total count of rate plans matching the filter.
  countRatePlans(filter: FilterRequest): Int!

  # Returns the latest version of a rate plan by its planId UUID.
  ratePlan(planId: ID!): RatePlan

  # Returns the latest version of each logical plan for the given planType.
  # For RETAIL, scopes to the tenant's wholesaleId (resolved from Host header).
  # Only intended for use from a RETAIL portal context.
  latestRatePlanList(planType: RatePlanType!): [RatePlan!]!
}

# ---------------------------------------------------------------------------
# Mutations
# ---------------------------------------------------------------------------

extend type Mutation {
  # Creates a new rate plan in DRAFT status with a new planId.
  # For RETAIL plans, wholesaleId is resolved from the tenant context (Host header).
  createRatePlan(ratePlan: RatePlanInput!): RatePlan!

  # Updates the name, type, effectiveAt, and rate lines of a DRAFT rate plan.
  # Returns an error if the plan version is not in DRAFT status.
  # TODO: confirm whether updateRatePlan or updateRatePlanRules is the one to keep.
  updateRatePlan(planId: ID!, ratePlan: RatePlanInput!): RatePlan!

  # Updates only the rate lines of a DRAFT rate plan, leaving metadata unchanged.
  # TODO: confirm whether updateRatePlan or updateRatePlanRules is the one to keep.
  updateRatePlanRules(planId: ID!, rateLines: [RateLineInput!]!): RatePlan!

  # Creates a new DRAFT version of an existing rate plan (same planId, new row).
  cloneRatePlan(planId: ID!): RatePlan!

  # Transitions a DRAFT rate plan version to PENDING (awaiting approval).
  submitRatePlanForApproval(planId: ID!): RatePlan!

  # Transitions a PENDING rate plan version to ACTIVE.
  # approvedBy is derived from the authenticated JWT.
  approveRatePlan(planId: ID!): RatePlan!

  # Transitions a PENDING rate plan version back to DRAFT.
  declineRatePlan(planId: ID!): RatePlan!

  # Permanently deletes the DRAFT version of a rate plan. Returns true on success.
  # Returns an error if the plan version is not in DRAFT status.
  deleteRatePlan(planId: ID!): Boolean!
}
```

---

## 24. RatePlanResource — SQL Queries

**File:** `internal/store/queries/rateplan.sql` — append to existing (keep `FindActiveRatePlans`).

```sql
-- name: FindLatestRatePlanByPlanId :one
-- Returns the most recent version of a logical rate plan by plan_id.
SELECT id, plan_id, modified_at, plan_type, wholesale_id, plan_name,
       rateplan, plan_status, created_by, approved_by, effective_at
FROM rateplan
WHERE plan_id = $1
ORDER BY effective_at DESC, id DESC
LIMIT 1;

-- name: CreateRatePlan :one
-- Inserts a new rate plan version in DRAFT status.
-- plan_id is generated by the application. wholesale_id is null for non-RETAIL plans.
INSERT INTO rateplan (plan_id, plan_type, wholesale_id, plan_name, rateplan, created_by, effective_at, plan_status, modified_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'DRAFT', NOW())
RETURNING id, plan_id, modified_at, plan_type, wholesale_id, plan_name,
          rateplan, plan_status, created_by, approved_by, effective_at;

-- name: UpdateRatePlan :one
-- Updates the name, type, effectiveAt, and rateplan JSONB of the DRAFT version of a logical plan.
-- Guards against modifying non-DRAFT versions.
UPDATE rateplan
SET plan_name    = $2,
    plan_type    = $3,
    effective_at = $4,
    rateplan     = $5,
    modified_at  = NOW()
WHERE plan_id    = $1
  AND plan_status = 'DRAFT'
RETURNING id, plan_id, modified_at, plan_type, wholesale_id, plan_name,
          rateplan, plan_status, created_by, approved_by, effective_at;

-- name: UpdateRatePlanRules :one
-- Updates only the rateplan JSONB of the DRAFT version of a logical plan.
-- TODO: confirm whether this or UpdateRatePlan is the one to keep in production.
UPDATE rateplan
SET rateplan    = $2,
    modified_at = NOW()
WHERE plan_id    = $1
  AND plan_status = 'DRAFT'
RETURNING id, plan_id, modified_at, plan_type, wholesale_id, plan_name,
          rateplan, plan_status, created_by, approved_by, effective_at;

-- name: SubmitRatePlan :one
-- Transitions the DRAFT version of a logical plan to PENDING.
UPDATE rateplan
SET plan_status = 'PENDING',
    modified_at = NOW()
WHERE plan_id    = $1
  AND plan_status = 'DRAFT'
RETURNING id, plan_id, modified_at, plan_type, wholesale_id, plan_name,
          rateplan, plan_status, created_by, approved_by, effective_at;

-- name: ApproveRatePlan :one
-- Transitions the PENDING version of a logical plan to ACTIVE and records the approver.
UPDATE rateplan
SET plan_status = 'ACTIVE',
    approved_by = $2,
    modified_at = NOW()
WHERE plan_id    = $1
  AND plan_status = 'PENDING'
RETURNING id, plan_id, modified_at, plan_type, wholesale_id, plan_name,
          rateplan, plan_status, created_by, approved_by, effective_at;

-- name: DeclineRatePlan :one
-- Transitions the PENDING version of a logical plan back to DRAFT, clearing the approver.
UPDATE rateplan
SET plan_status = 'DRAFT',
    approved_by = NULL,
    modified_at = NOW()
WHERE plan_id    = $1
  AND plan_status = 'PENDING'
RETURNING id, plan_id, modified_at, plan_type, wholesale_id, plan_name,
          rateplan, plan_status, created_by, approved_by, effective_at;

-- name: DeleteRatePlan :exec
-- Permanently deletes the DRAFT version of a logical plan.
DELETE FROM rateplan
WHERE plan_id    = $1
  AND plan_status = 'DRAFT';

-- name: LatestRatePlanByType :many
-- Returns the most recent version of each logical plan for the given planType.
-- For RETAIL plans pass the tenant's wholesale_id; for others pass NULL.
SELECT DISTINCT ON (plan_id)
    id, plan_id, modified_at, plan_type, wholesale_id, plan_name,
    rateplan, plan_status, created_by, approved_by, effective_at
FROM rateplan
WHERE plan_type = $1
  AND ($2::uuid IS NULL OR wholesale_id = $2)
ORDER BY plan_id, effective_at DESC, id DESC;
```

**File:** `internal/store/queries/wholesaler.sql` — new file:

```sql
-- name: AllWholesalers :many
-- Returns all active wholesaler records. Used by the tenant resolver to build
-- the hostname → wholesale_id lookup map.
SELECT id, hosts FROM wholesaler WHERE active = true;
```

> **Note on `query_parameter_limit: 4`:** `CreateRatePlan` has 7 params and will generate a
> `CreateRatePlanParams` struct. `UpdateRatePlan` has 5 params → `UpdateRatePlanParams` struct.
> All other queries have ≤ 4 params and use individual arguments.

---

## 25. RatePlanResource — Store Layer

**File:** `internal/store/rateplan_store.go` — new hand-written file.

```go
type ListRatePlansParams struct {
    WhereSQL string
    Args     []any
    OrderSQL string
    Limit    int
    Offset   int
}

// ListRatePlans executes a dynamic rate plan query (all versions, no DISTINCT ON).
func (s *Store) ListRatePlans(ctx context.Context, p ListRatePlansParams) ([]sqlc.Rateplan, error)

// CountRatePlans executes a dynamic count query.
func (s *Store) CountRatePlans(ctx context.Context, whereSQL string, args []any) (int64, error)
```

Wildcard columns (mirrors Java `RatePlanEntity.WILDCARD_FIELDS`):
`plan_id`, `plan_name`, `plan_type`, `plan_status`

Column map:
```go
var ratePlanColumns = map[string]string{
    "planId":     "plan_id",
    "planName":   "plan_name",
    "planType":   "plan_type",
    "planStatus": "plan_status",
    "createdBy":  "created_by",
    "approvedBy": "approved_by",
    "effectiveAt":"effective_at",
    "modifiedAt": "modified_at",
    "wholesaleId":"wholesale_id",
}
```

---

## 26. RatePlanResource — Service Layer

**File:** `internal/backend/services/rateplan_service.go` — new file.

### 26.1 Service Methods

```go
type RatePlanService struct { store *store.Store }

func NewRatePlanService(s *store.Store) *RatePlanService

func (s *RatePlanService) ListRatePlans(ctx, page, filter) ([]*model.RatePlan, error)
func (s *RatePlanService) CountRatePlans(ctx, filter) (int, error)
func (s *RatePlanService) GetRatePlan(ctx, planId string) (*model.RatePlan, error)
func (s *RatePlanService) LatestRatePlanList(ctx, planType model.RatePlanType) ([]*model.RatePlan, error)
func (s *RatePlanService) CreateRatePlan(ctx, input model.RatePlanInput) (*model.RatePlan, error)
func (s *RatePlanService) UpdateRatePlan(ctx, planId string, input model.RatePlanInput) (*model.RatePlan, error)
func (s *RatePlanService) UpdateRatePlanRules(ctx, planId string, rateLines []*model.RateLineInput) (*model.RatePlan, error)
func (s *RatePlanService) CloneRatePlan(ctx, planId string) (*model.RatePlan, error)
func (s *RatePlanService) SubmitRatePlanForApproval(ctx, planId string) (*model.RatePlan, error)
func (s *RatePlanService) ApproveRatePlan(ctx, planId string) (*model.RatePlan, error)
func (s *RatePlanService) DeclineRatePlan(ctx, planId string) (*model.RatePlan, error)
func (s *RatePlanService) DeleteRatePlan(ctx, planId string) (bool, error)
```

### 26.2 Key Mapping Decisions

| Mapping | Detail |
|---|---|
| `sqlc.Rateplan.Rateplan []byte` → `model.RatePlan` | `json.Unmarshal` |
| `model.RatePlan` → `graphqlmodel.RatePlan` | Field-by-field; rateLines converted |
| `charging.RateKey` → `String` | `rk.String()` (dot-separated) |
| `String` → `charging.RateKey` | `charging.ParseRateKey(s)` |
| `decimal.Decimal` → `String` | `.String()` |
| `String` → `decimal.Decimal` | `decimal.NewFromString(s)` |
| `model.Quantity` → `Int` | `int(q)` |
| `Int` → `model.Quantity` | `model.Quantity(n)` |
| `pgtype.UUID wholesale_id` → `*String` | nil if not valid |
| RETAIL create | Read `tenant.WholesaleIDFromContext(ctx)`; error if missing |

### 26.3 JSONB Sync on Create/Update

When building the `model.RatePlan` struct for the JSONB column, the service also sets:
```go
plan.RatePlanID   = planId          // new UUID string
plan.RatePlanName = input.PlanName
plan.RatePlanType = model.RatePlanType(input.PlanType)
plan.EffectiveFrom = effectiveAt.Time
```
This ensures the charging engine always reads a self-consistent embedded document.

---

## 27. RatePlanResource — Resolver Layer

**File:** `internal/backend/resolvers/rateplan.resolvers.go`

All methods delegate to `RatePlanService` — zero business logic in the resolver.

**`resolver.go` addition:**
```go
type Resolver struct {
    CarrierSvc        *services.CarrierService
    ClassificationSvc *services.ClassificationService
    RatePlanSvc       *services.RatePlanService   // ADD
}
```

---

## 28. RatePlanResource — File Map

| File | Action | Notes |
|---|---|---|
| `internal/store/queries/wholesaler.sql` | **New** | `AllWholesalers` query for tenant resolver |
| `internal/store/queries/rateplan.sql` | **Append** | 9 new sqlc queries |
| `internal/store/sqlc/wholesaler.sql.go` | **New (generated)** | `sqlc generate` |
| `internal/store/sqlc/rateplan.sql.go` | **New (generated)** | `sqlc generate` |
| `internal/backend/appcontext/config.go` | **Modify** | Add `TenantRefreshInterval` to `ServerConfig` |
| `internal/auth/tenant/resolver.go` | **New** | In-memory host→wholesaleId map with background refresh |
| `internal/auth/tenant/middleware.go` | **New** | HTTP middleware; `WholesaleIDFromContext` |
| `gql/schema/rateplan.graphql` | **New** | Full schema: enums, types, inputs, queries, mutations |
| `internal/store/sqlc/generated (models.go)` | **Regenerate** | `sqlc generate` (no changes needed) |
| `internal/backend/graphql/generated/generated.go` | **Regenerate** | `gqlgen generate` |
| `internal/backend/graphql/model/models_gen.go` | **Regenerate** | `gqlgen generate` |
| `internal/store/rateplan_store.go` | **New** | `ListRatePlans`, `CountRatePlans` dynamic pgx |
| `internal/backend/services/rateplan_service.go` | **New** | Full `RatePlanService` implementation |
| `internal/backend/resolvers/rateplan.resolvers.go` | **New** | Thin resolver delegation |
| `internal/backend/resolvers/resolver.go` | **Modify** | Add `RatePlanSvc` |
| `internal/backend/appcontext/context.go` | **Modify** | Add `TenantResolver`, `RatePlanSvc` |
| `internal/backend/handlers/graphql/router.go` | **Modify** | Add tenant middleware; pass `RatePlanSvc` |
| `cmd/charging-backend/backend-config.yaml` | **Modify** | Add `tenantRefreshInterval` |
| `internal/backend/services/rateplan_service_test.go` | **New** | Unit tests |
| `RatePlanGraphQL.http` | **New** | HTTP test file for all queries and mutations |
