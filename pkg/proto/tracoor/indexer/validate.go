package indexer

import (
	"errors"
	"fmt"
)

func (s *BeaconState) Validate() error {
	if s == nil {
		return errors.New("beacon state is nil")
	}

	if s.GetStateRoot().Value == "" {
		return errors.New("root is required")
	}

	if s.GetEpoch() == nil {
		return errors.New("epoch is required")
	}

	if s.GetSlot() == nil {
		return errors.New("slot is required")
	}

	if s.GetStateRoot().Value == "" {
		return errors.New("state root is required")
	}

	if s.GetId() == nil {
		return errors.New("id is required")
	}

	return nil
}

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

func (r *ListUniqueBeaconStateValuesRequest) Validate() error {
	if r == nil {
		return errors.New("request is nil")
	}

	if len(r.Fields) == 0 {
		return errors.New("fields is required")
	}

	return nil
}
