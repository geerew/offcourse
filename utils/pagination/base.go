package pagination

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strconv"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	DefaultPerPage    int    = 30
	MaxPerPage        int    = 500
	PageQueryParam    string = "page"
	PerPageQueryParam string = "perPage"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// PaginationResult defines the return result of a pagination
type PaginationResult struct {
	Page       int               `json:"page"`
	PerPage    int               `json:"perPage"`
	TotalItems int               `json:"totalItems"`
	TotalPages int               `json:"totalPages"`
	Items      []json.RawMessage `json:"items"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Pagination holds page/limit state for offset-based list queries
type Pagination struct {
	page       int
	perPage    int
	totalItems int
	totalPages int
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// New creates a pagination with normalized page and perPage values
func New(page, perPage int) *Pagination {
	if page <= 0 {
		page = 1
	}

	if perPage <= 0 {
		perPage = DefaultPerPage
	} else if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	return &Pagination{
		page:    page,
		perPage: perPage,
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ParsePage normalizes a page query value (defaults to 1)
func ParsePage(s string) int {
	res, err := strconv.Atoi(s)
	if err != nil || res <= 0 {
		return 1
	}

	return res
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ParsePerPage normalizes a perPage query value (defaults to DefaultPerPage, capped at
// MaxPerPage)
func ParsePerPage(s string) int {
	res, err := strconv.Atoi(s)
	if err != nil || res <= 0 {
		return DefaultPerPage
	}

	if res > MaxPerPage {
		return MaxPerPage
	}

	return res
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Limit returns the limit value
func (p *Pagination) Limit() int {
	return p.perPage
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Offset calculates and return an offset value
func (p *Pagination) Offset() int {
	return p.perPage * (p.page - 1)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// TotalItems returns the total number of items
func (p *Pagination) TotalItems() int {
	return p.totalItems
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// TotalPages returns the total number of pages
func (p *Pagination) TotalPages() int {
	return p.totalPages
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// SetCount sets the total number of items and calculates the total number of pages
func (p *Pagination) SetCount(count int) {
	p.totalItems = count
	p.totalPages = int(math.Ceil(float64(p.totalItems) / float64(p.perPage)))
}

// BuildResult builds a result object from the pagination values, which is suitable for
// a HTTP response
func (p *Pagination) BuildResult(m any) (*PaginationResult, error) {
	items := []json.RawMessage{}

	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Slice {
		return nil, errors.New("input is not a slice")
	}

	for i := 0; i < v.Len(); i++ {
		raw, err := json.Marshal(v.Index(i).Interface())
		if err != nil {
			return nil, err
		}
		items = append(items, raw)
	}

	return &PaginationResult{
		Page:       p.page,
		PerPage:    p.perPage,
		TotalItems: p.totalItems,
		TotalPages: p.totalPages,
		Items:      items,
	}, nil
}
