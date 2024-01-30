package persistence

import (
	"errors"

	"gorm.io/gorm"
)

type PaginationCursor struct {
	// The cursor to start from.
	Offset int `json:"offset"`
	// The number of items to return.
	Limit int `json:"limit"`
	// OrderBy is the column to order by.
	OrderBy string `json:"order_by"`
}

func (p *PaginationCursor) ApplyOffsetLimit(query *gorm.DB) *gorm.DB {
	if p.Limit != 0 {
		query = query.Limit(p.Limit)
	}

	return query.Offset(p.Offset)
}

func (p *PaginationCursor) ApplyOrderBy(query *gorm.DB) *gorm.DB {
	if p.OrderBy != "" {
		query = query.Order(p.OrderBy)
	} else {
		query = query.Order("fetched_at ASC")
	}

	return query
}

func (p *PaginationCursor) Validate() error {
	if p.Limit < 0 {
		return errors.New("invalid limit")
	}

	if p.Offset < 0 {
		return errors.New("invalid offset")
	}

	return nil
}
