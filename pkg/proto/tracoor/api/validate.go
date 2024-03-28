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

func (r *ListBeaconBlockRequest) Validate() error {
	if r.Pagination != nil {
		if err := r.Pagination.Validate(); err != nil {
			return fmt.Errorf("invalid pagination: %w", err)
		}
	}

	return nil
}

func (r *ListBeaconBadBlockRequest) Validate() error {
	if r.Pagination != nil {
		if err := r.Pagination.Validate(); err != nil {
			return fmt.Errorf("invalid pagination: %w", err)
		}
	}

	return nil
}

func (r *ListBeaconBadBlobRequest) Validate() error {
	if r.Pagination != nil {
		if err := r.Pagination.Validate(); err != nil {
			return fmt.Errorf("invalid pagination: %w", err)
		}
	}

	return nil
}

func (r *ListExecutionBlockTraceRequest) Validate() error {
	if r.Pagination != nil {
		if err := r.Pagination.Validate(); err != nil {
			return fmt.Errorf("invalid pagination: %w", err)
		}
	}

	return nil
}

func (r *ListExecutionBadBlockRequest) Validate() error {
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

func (r *ListUniqueBeaconStateValuesRequest) Validate() error {
	if r == nil {
		return errors.New("request is nil")
	}

	if len(r.Fields) == 0 {
		return errors.New("fields is required")
	}

	return nil
}

func (r *ListUniqueBeaconBlockValuesRequest) Validate() error {
	if r == nil {
		return errors.New("request is nil")
	}

	if len(r.Fields) == 0 {
		return errors.New("fields is required")
	}

	return nil
}

func (r *ListUniqueBeaconBadBlockValuesRequest) Validate() error {
	if r == nil {
		return errors.New("request is nil")
	}

	if len(r.Fields) == 0 {
		return errors.New("fields is required")
	}

	return nil
}

func (r *ListUniqueBeaconBadBlobValuesRequest) Validate() error {
	if r == nil {
		return errors.New("request is nil")
	}

	if len(r.Fields) == 0 {
		return errors.New("fields is required")
	}

	return nil
}

func (r *ListUniqueExecutionBlockTraceValuesRequest) Validate() error {
	if r == nil {
		return errors.New("request is nil")
	}

	if len(r.Fields) == 0 {
		return errors.New("fields is required")
	}

	return nil
}

func (r *ListUniqueExecutionBadBlockValuesRequest) Validate() error {
	if r == nil {
		return errors.New("request is nil")
	}

	if len(r.Fields) == 0 {
		return errors.New("fields is required")
	}

	return nil
}
