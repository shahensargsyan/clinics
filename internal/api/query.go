package api

import (
	"strings"

	"gorm.io/gorm"
)

const (
	defaultPerPage = 25
	maxPerPage     = 100
)

// pageOpts is the normalized form of the page/per_page/search/sort
// quartet. Handlers should always go through normalize() rather than
// reading the raw *T params, so out-of-range and malformed values are
// clamped/parsed in one place.
type pageOpts struct {
	page    int
	perPage int
	sortCol string // "" when no sort requested
	sortDir string // "asc" or "desc"; meaningful only when sortCol != ""
	search  string
}

// normalize takes the four pointer fields every list endpoint shares
// (page/per_page/search/sort) and clamps them to the documented ranges.
// Passing primitives lets every resource's strict ListParams struct call
// in without us having to define one signature per resource.
func normalize(page, perPage *int, search, sort *string) pageOpts {
	opts := pageOpts{page: 1, perPage: defaultPerPage}
	if page != nil && *page > 0 {
		opts.page = *page
	}
	if perPage != nil {
		switch {
		case *perPage < 1:
			opts.perPage = defaultPerPage
		case *perPage > maxPerPage:
			opts.perPage = maxPerPage
		default:
			opts.perPage = *perPage
		}
	}
	if search != nil {
		opts.search = strings.TrimSpace(*search)
	}
	if sort != nil {
		s := strings.TrimSpace(*sort)
		switch {
		case strings.HasPrefix(s, "-"):
			opts.sortCol = s[1:]
			opts.sortDir = "desc"
		case s != "":
			opts.sortCol = s
			opts.sortDir = "asc"
		}
	}
	return opts
}

// applySearch adds a `WHERE col1 LIKE ? OR col2 LIKE ? ...` clause across
// the caller-supplied column list. Match is case-insensitive (LOWER on
// both sides) and substring (%needle%), matching Backpack's default search
// behaviour. Column names are interpolated directly into the SQL string;
// callers MUST pass a Go-defined whitelist, never user input.
func applySearch(q *gorm.DB, search string, columns []string) *gorm.DB {
	if search == "" || len(columns) == 0 {
		return q
	}
	needle := "%" + strings.ToLower(search) + "%"
	conds := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns))
	for _, col := range columns {
		conds = append(conds, "LOWER(`"+col+"`) LIKE ?")
		args = append(args, needle)
	}
	return q.Where(strings.Join(conds, " OR "), args...)
}

// applySort adds ORDER BY when sortCol is in the whitelist, falling back
// to `id DESC` (Backpack's default) otherwise. As with applySearch, the
// column name is interpolated, so `allowed` must be a Go-defined map.
func applySort(q *gorm.DB, sortCol, sortDir string, allowed map[string]struct{}) *gorm.DB {
	if _, ok := allowed[sortCol]; !ok {
		return q.Order("id DESC")
	}
	dir := "ASC"
	if strings.EqualFold(sortDir, "desc") {
		dir = "DESC"
	}
	return q.Order("`" + sortCol + "` " + dir)
}

// applyPaginate adds LIMIT/OFFSET and returns the PaginationMeta to embed
// in the response envelope. `total` should already have been computed via
// a separate Count() on the search-filtered (but unsorted, unpaginated)
// query.
func applyPaginate(q *gorm.DB, page, perPage int, total int64) (*gorm.DB, PaginationMeta) {
	lastPage := int((total + int64(perPage) - 1) / int64(perPage))
	if lastPage < 1 {
		lastPage = 1
	}
	return q.Limit(perPage).Offset((page - 1) * perPage), PaginationMeta{
		CurrentPage: page,
		PerPage:     perPage,
		Total:       total,
		LastPage:    lastPage,
	}
}
