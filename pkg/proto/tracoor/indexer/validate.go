package indexer

import (
	"errors"
	"fmt"
)

func (req *CreateBeaconStateRequest) Validate() error {
	if req.GetLocation().GetValue() == "" {
		return fmt.Errorf("location is required")
	}

	if req.GetNode().GetValue() == "" {
		return fmt.Errorf("node is required")
	}

	if req.Epoch == nil {
		return fmt.Errorf("epoch is required")
	}

	if req.Slot == nil {
		return fmt.Errorf("slot is required")
	}

	if req.GetStateRoot().Value == "" {
		return fmt.Errorf("state root is required")
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
