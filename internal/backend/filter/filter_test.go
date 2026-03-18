package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-ocs/internal/backend/graphql/model"
)

// allowedCols is a small allowlist shared across filter tests.
var testCols = map[string]string{
	"name":    "carrier_name",
	"country": "country_name",
	"iso":     "iso",
}

var wildcardCols = []string{"carrier_name", "country_name"}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

// --- BuildWhere ---

func TestBuildWhere_NilRequest(t *testing.T) {
	wc, err := BuildWhere(nil, testCols, wildcardCols)
	require.NoError(t, err)
	assert.Empty(t, wc.SQL)
	assert.Nil(t, wc.Args)
}

func TestBuildWhere_NoFiltersNoWildcard(t *testing.T) {
	req := &model.FilterRequest{}
	wc, err := BuildWhere(req, testCols, wildcardCols)
	require.NoError(t, err)
	assert.Empty(t, wc.SQL)
	assert.Nil(t, wc.Args)
}

func TestBuildWhere_SingleEqualsFilter(t *testing.T) {
	req := &model.FilterRequest{
		Filters: []*model.FilterInput{
			{Key: "iso", Operation: "=", Value: "NZ"},
		},
	}
	wc, err := BuildWhere(req, testCols, wildcardCols)
	require.NoError(t, err)
	assert.Equal(t, "WHERE iso = $1", wc.SQL)
	require.Len(t, wc.Args, 1)
	assert.Equal(t, "NZ", wc.Args[0])
}

func TestBuildWhere_MultipleFilters(t *testing.T) {
	req := &model.FilterRequest{
		Filters: []*model.FilterInput{
			{Key: "iso", Operation: "=", Value: "NZ"},
			{Key: "name", Operation: "ILIKE", Value: "%Vodafone%"},
		},
	}
	wc, err := BuildWhere(req, testCols, wildcardCols)
	require.NoError(t, err)
	assert.Contains(t, wc.SQL, "WHERE")
	assert.Contains(t, wc.SQL, "AND")
	assert.Len(t, wc.Args, 2)
}

func TestBuildWhere_INOperator(t *testing.T) {
	req := &model.FilterRequest{
		Filters: []*model.FilterInput{
			{Key: "iso", Operation: "IN", Value: "NZ,AU,GB"},
		},
	}
	wc, err := BuildWhere(req, testCols, wildcardCols)
	require.NoError(t, err)
	assert.Contains(t, wc.SQL, "IN ($1, $2, $3)")
	assert.Len(t, wc.Args, 3)
	assert.Equal(t, "NZ", wc.Args[0])
	assert.Equal(t, "AU", wc.Args[1])
	assert.Equal(t, "GB", wc.Args[2])
}

func TestBuildWhere_NOTINOperator(t *testing.T) {
	req := &model.FilterRequest{
		Filters: []*model.FilterInput{
			{Key: "iso", Operation: "NOT IN", Value: "US,CA"},
		},
	}
	wc, err := BuildWhere(req, testCols, wildcardCols)
	require.NoError(t, err)
	assert.Contains(t, wc.SQL, "NOT IN ($1, $2)")
	assert.Len(t, wc.Args, 2)
}

func TestBuildWhere_WildcardOnly(t *testing.T) {
	req := &model.FilterRequest{Wildcard: strPtr("vodafone")}
	wc, err := BuildWhere(req, testCols, wildcardCols)
	require.NoError(t, err)
	assert.Contains(t, wc.SQL, "WHERE")
	assert.Contains(t, wc.SQL, "ILIKE $1")
	assert.Contains(t, wc.SQL, "OR")
	require.Len(t, wc.Args, 1)
	assert.Equal(t, "%vodafone%", wc.Args[0])
}

func TestBuildWhere_WildcardAndFilter(t *testing.T) {
	req := &model.FilterRequest{
		Filters:  []*model.FilterInput{{Key: "iso", Operation: "=", Value: "NZ"}},
		Wildcard: strPtr("voda"),
	}
	wc, err := BuildWhere(req, testCols, wildcardCols)
	require.NoError(t, err)
	assert.Equal(t, 2, len(wc.Args))
	assert.Equal(t, "NZ", wc.Args[0])
	assert.Equal(t, "%voda%", wc.Args[1])
}

func TestBuildWhere_WildcardEmptyString_Ignored(t *testing.T) {
	req := &model.FilterRequest{Wildcard: strPtr("   ")}
	wc, err := BuildWhere(req, testCols, wildcardCols)
	require.NoError(t, err)
	assert.Empty(t, wc.SQL)
}

func TestBuildWhere_InvalidKey(t *testing.T) {
	req := &model.FilterRequest{
		Filters: []*model.FilterInput{
			{Key: "secret", Operation: "=", Value: "x"},
		},
	}
	_, err := BuildWhere(req, testCols, wildcardCols)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret")
}

func TestBuildWhere_InvalidOperation(t *testing.T) {
	req := &model.FilterRequest{
		Filters: []*model.FilterInput{
			{Key: "iso", Operation: "DROP TABLE", Value: "x"},
		},
	}
	_, err := BuildWhere(req, testCols, wildcardCols)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not permitted")
}

func TestBuildWhere_OperationCaseInsensitive(t *testing.T) {
	req := &model.FilterRequest{
		Filters: []*model.FilterInput{
			{Key: "iso", Operation: "ilike", Value: "nz%"},
		},
	}
	_, err := BuildWhere(req, testCols, wildcardCols)
	require.NoError(t, err)
}

// --- BuildOrderBy ---

func TestBuildOrderBy_NilRequest_UsesDefault(t *testing.T) {
	order, err := BuildOrderBy(nil, "plmn", testCols)
	require.NoError(t, err)
	assert.Equal(t, "ORDER BY plmn ASC", order)
}

func TestBuildOrderBy_NoSort_UsesDefault(t *testing.T) {
	order, err := BuildOrderBy(&model.PageRequest{}, "plmn", testCols)
	require.NoError(t, err)
	assert.Equal(t, "ORDER BY plmn ASC", order)
}

func TestBuildOrderBy_ValidKeyASC(t *testing.T) {
	req := &model.PageRequest{Sort: &model.SortInput{Key: "name", Order: "asc"}}
	order, err := BuildOrderBy(req, "plmn", testCols)
	require.NoError(t, err)
	assert.Equal(t, "ORDER BY carrier_name ASC", order)
}

func TestBuildOrderBy_ValidKeyDESC(t *testing.T) {
	req := &model.PageRequest{Sort: &model.SortInput{Key: "iso", Order: "DESC"}}
	order, err := BuildOrderBy(req, "plmn", testCols)
	require.NoError(t, err)
	assert.Equal(t, "ORDER BY iso DESC", order)
}

func TestBuildOrderBy_InvalidKey(t *testing.T) {
	req := &model.PageRequest{Sort: &model.SortInput{Key: "badField", Order: "asc"}}
	_, err := BuildOrderBy(req, "plmn", testCols)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "badField")
}

// --- PageOffset ---

func TestPageOffset_NilRequest_Defaults(t *testing.T) {
	limit, offset := PageOffset(nil)
	assert.Equal(t, 10, limit)
	assert.Equal(t, 0, offset)
}

func TestPageOffset_EmptyRequest_Defaults(t *testing.T) {
	limit, offset := PageOffset(&model.PageRequest{})
	assert.Equal(t, 10, limit)
	assert.Equal(t, 0, offset)
}

func TestPageOffset_CustomPageSize(t *testing.T) {
	limit, offset := PageOffset(&model.PageRequest{PageSize: intPtr(25)})
	assert.Equal(t, 25, limit)
	assert.Equal(t, 0, offset)
}

func TestPageOffset_SecondPage(t *testing.T) {
	limit, offset := PageOffset(&model.PageRequest{PageSize: intPtr(20), PageIndex: intPtr(2)})
	assert.Equal(t, 20, limit)
	assert.Equal(t, 40, offset) // page 2 * 20 = 40
}

func TestPageOffset_ZeroPageSize_UsesDefault(t *testing.T) {
	limit, offset := PageOffset(&model.PageRequest{PageSize: intPtr(0)})
	assert.Equal(t, 10, limit)
	assert.Equal(t, 0, offset)
}

func TestPageOffset_ZeroPageIndex_IsFirstPage(t *testing.T) {
	limit, offset := PageOffset(&model.PageRequest{PageSize: intPtr(5), PageIndex: intPtr(0)})
	assert.Equal(t, 5, limit)
	assert.Equal(t, 0, offset)
}
