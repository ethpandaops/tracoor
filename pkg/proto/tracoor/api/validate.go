package api

import (
	"errors"
	"fmt"
)

func (r *ListBeaconStateRequest) Validate() error {
	if r.Pagination != nil {
		if err := r.Pagination.Validate(); err != nil {
			return fmt.Errorf("invalid pagination: %w", err)
		}
	}

	return nil
}

func (r *PaginationCursor) Validate() error {
	if r.Limit == 0 {
		return errors.New("limit is required")
	}

	return nil
}

func (r *ListUniqueValuesRequest) Validate() error {
	if r == nil {
		return errors.New("request is nil")
	}

	if r.Entity == Entity_UNKNOWN {
		return errors.New("entity is required")
	}

	if len(r.Fields) == 0 {
		return errors.New("fields is required")
	}

	return nil
}
