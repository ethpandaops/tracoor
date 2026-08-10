package persistence

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// DefaultPageLimit is used when a caller supplies no limit of its own.
	DefaultPageLimit = 100
	// MaxPageLimit is the hard ceiling on rows a single list query may return. Retention
	// asks for exactly this many, so it must not be lowered without changing that caller.
	MaxPageLimit = 10000
)

// sortableColumns is the union of columns the persisted models can be ordered by. GORM treats
// an order string as raw SQL, so anything outside this set is rejected before it reaches the
// query builder.
var sortableColumns = map[string]struct{}{
	"id":                       {},
	"node":                     {},
	"slot":                     {},
	"epoch":                    {},
	"index":                    {},
	"state_root":               {},
	"block_root":               {},
	"block_hash":               {},
	"block_number":             {},
	"block_extra_data":         {},
	"fetched_at":               {},
	"observed_at":              {},
	"network":                  {},
	"location":                 {},
	"content_encoding":         {},
	"node_version":             {},
	"beacon_implementation":    {},
	"execution_implementation": {},
}

var defaultOrderBy = clause.OrderByColumn{
	Column: clause.Column{Name: "fetched_at"},
	Desc:   true,
}

//nolint:tagliatelle // requires snake.
type PaginationCursor struct {
	// The cursor to start from.
	Offset int `json:"offset"`
	// The number of items to return.
	Limit int `json:"limit"`
	// OrderBy is the column to order by.
	OrderBy string `json:"order_by"`
}

func (p *PaginationCursor) ApplyOffsetLimit(query *gorm.DB) *gorm.DB {
	limit := p.Limit
	if limit <= 0 {
		limit = DefaultPageLimit
	}

	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}

	offset := p.Offset
	if offset < 0 {
		offset = 0
	}

	return query.Limit(limit).Offset(offset)
}

func (p *PaginationCursor) ApplyOrderBy(query *gorm.DB) (*gorm.DB, error) {
	if p.OrderBy == "" {
		return query.Order(defaultOrderBy), nil
	}

	order, err := parseOrderBy(p.OrderBy)
	if err != nil {
		return nil, err
	}

	return query.Order(order), nil
}

func (p *PaginationCursor) Validate() error {
	if p.Limit < 0 {
		return errors.New("invalid limit")
	}

	if p.Offset < 0 {
		return errors.New("invalid offset")
	}

	if p.OrderBy != "" {
		if _, err := parseOrderBy(p.OrderBy); err != nil {
			return err
		}
	}

	return nil
}

// parseOrderBy accepts "{column}" or "{column} {ASC|DESC}" and nothing else. The column is
// returned as a quoted identifier rather than raw SQL.
func parseOrderBy(orderBy string) (clause.OrderByColumn, error) {
	parts := strings.Fields(orderBy)

	var desc bool

	switch len(parts) {
	case 1:
	case 2:
		switch strings.ToUpper(parts[1]) {
		case "ASC":
		case "DESC":
			desc = true
		default:
			return clause.OrderByColumn{}, fmt.Errorf("invalid order by direction: %q", parts[1])
		}
	default:
		return clause.OrderByColumn{}, fmt.Errorf("invalid order by: %q", orderBy)
	}

	column := strings.ToLower(parts[0])

	if _, ok := sortableColumns[column]; !ok {
		return clause.OrderByColumn{}, fmt.Errorf("invalid order by column: %q", parts[0])
	}

	return clause.OrderByColumn{
		Column: clause.Column{Name: column},
		Desc:   desc,
	}, nil
}
