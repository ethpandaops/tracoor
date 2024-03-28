package indexer

import (
	"errors"
	"fmt"
)

func (s *BeaconState) Validate() error {
	if s == nil {
		return errors.New("beacon state is nil")
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

	if s.GetBeaconImplementation().GetValue() == "" {
		return errors.New("beacon implementation is required")
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

	if req.GetBeaconImplementation().Value == "" {
		return fmt.Errorf("beacon implementation is required")
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

func (s *BeaconBlock) Validate() error {
	if s == nil {
		return errors.New("bad  is nil")
	}

	if s.GetEpoch() == nil {
		return errors.New("epoch is required")
	}

	if s.GetSlot() == nil {
		return errors.New("slot is required")
	}

	if s.GetBlockRoot().Value == "" {
		return errors.New("block root is required")
	}

	if s.GetId() == nil {
		return errors.New("id is required")
	}

	if s.GetBeaconImplementation().GetValue() == "" {
		return errors.New("beacon implementation is required")
	}

	return nil
}

func (req *CreateBeaconBlockRequest) Validate() error {
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

	if req.GetBlockRoot().Value == "" {
		return fmt.Errorf("block root is required")
	}

	if req.GetBeaconImplementation().Value == "" {
		return fmt.Errorf("beacon implementation is required")
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

func (s *BeaconBadBlock) Validate() error {
	if s == nil {
		return errors.New("beacon bad block is nil")
	}

	if s.GetEpoch() == nil {
		return errors.New("epoch is required")
	}

	if s.GetSlot() == nil {
		return errors.New("slot is required")
	}

	if s.GetBlockRoot().Value == "" {
		return errors.New("block root is required")
	}

	if s.GetId() == nil {
		return errors.New("id is required")
	}

	if s.GetBeaconImplementation().GetValue() == "" {
		return errors.New("beacon implementation is required")
	}

	return nil
}

func (req *CreateBeaconBadBlockRequest) Validate() error {
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

	if req.GetBlockRoot().Value == "" {
		return fmt.Errorf("block root is required")
	}

	if req.GetBeaconImplementation().Value == "" {
		return fmt.Errorf("beacon implementation is required")
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

func (s *BeaconBadBlob) Validate() error {
	if s == nil {
		return errors.New("beacon bad block is nil")
	}

	if s.GetEpoch() == nil {
		return errors.New("epoch is required")
	}

	if s.GetSlot() == nil {
		return errors.New("slot is required")
	}

	if s.GetBlockRoot().Value == "" {
		return errors.New("block root is required")
	}

	if s.GetId() == nil {
		return errors.New("id is required")
	}

	if s.GetBeaconImplementation().GetValue() == "" {
		return errors.New("beacon implementation is required")
	}

	if s.GetIndex() == nil {
		return errors.New("index is required")
	}

	return nil
}

func (req *CreateBeaconBadBlobRequest) Validate() error {
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

	if req.GetBlockRoot().Value == "" {
		return fmt.Errorf("block root is required")
	}

	if req.GetBeaconImplementation().Value == "" {
		return fmt.Errorf("beacon implementation is required")
	}

	if req.Index == nil {
		return fmt.Errorf("index is required")
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

func (req *GetStorageHandshakeTokenRequest) Validate() error {
	if req.GetNode() == "" {
		return fmt.Errorf("node is required")
	}

	if req.GetToken() == "" {
		return fmt.Errorf("token is required")
	}

	return nil
}

func (req *CreateExecutionBlockTraceRequest) Validate() error {
	if req.GetLocation().GetValue() == "" {
		return fmt.Errorf("location is required")
	}

	if req.GetNode().GetValue() == "" {
		return fmt.Errorf("node is required")
	}

	if req.GetBlockHash().GetValue() == "" {
		return fmt.Errorf("block_hash is required")
	}

	if req.GetBlockNumber() == nil {
		return fmt.Errorf("block_number is required")
	}

	if req.GetExecutionImplementation().GetValue() == "" {
		return fmt.Errorf("execution_implementation is required")
	}

	if req.GetNodeVersion().GetValue() == "" {
		return fmt.Errorf("node_version is required")
	}

	if req.GetNetwork().GetValue() == "" {
		return fmt.Errorf("network is required")
	}

	return nil
}

func (t *ExecutionBlockTrace) Validate() error {
	if t == nil {
		return errors.New("execution block trace is nil")
	}

	if t.GetLocation().GetValue() == "" {
		return fmt.Errorf("location is required")
	}

	if t.GetNode().GetValue() == "" {
		return fmt.Errorf("node is required")
	}

	if t.GetBlockHash().GetValue() == "" {
		return fmt.Errorf("block_hash is required")
	}

	if t.GetBlockNumber() == nil {
		return fmt.Errorf("block_number is required")
	}

	if t.GetExecutionImplementation().GetValue() == "" {
		return fmt.Errorf("execution_implementation is required")
	}

	if t.GetNodeVersion().GetValue() == "" {
		return fmt.Errorf("node_version is required")
	}

	if t.GetNetwork().GetValue() == "" {
		return fmt.Errorf("network is required")
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

func (req *CreateExecutionBadBlockRequest) Validate() error {
	if req.GetLocation().GetValue() == "" {
		return fmt.Errorf("location is required")
	}

	if req.GetNode().GetValue() == "" {
		return fmt.Errorf("node is required")
	}

	if req.GetBlockHash().GetValue() == "" {
		return fmt.Errorf("block_hash is required")
	}

	if req.GetBlockNumber() == nil {
		return fmt.Errorf("block_number is required")
	}

	if req.GetExecutionImplementation().GetValue() == "" {
		return fmt.Errorf("execution_implementation is required")
	}

	if req.GetNodeVersion().GetValue() == "" {
		return fmt.Errorf("node_version is required")
	}

	if req.GetNetwork().GetValue() == "" {
		return fmt.Errorf("network is required")
	}

	return nil
}

func (t *ExecutionBadBlock) Validate() error {
	if t == nil {
		return errors.New("execution block trace is nil")
	}

	if t.GetLocation().GetValue() == "" {
		return fmt.Errorf("location is required")
	}

	if t.GetNode().GetValue() == "" {
		return fmt.Errorf("node is required")
	}

	if t.GetBlockHash().GetValue() == "" {
		return fmt.Errorf("block_hash is required")
	}

	if t.GetExecutionImplementation().GetValue() == "" {
		return fmt.Errorf("execution_implementation is required")
	}

	if t.GetNodeVersion().GetValue() == "" {
		return fmt.Errorf("node_version is required")
	}

	if t.GetNetwork().GetValue() == "" {
		return fmt.Errorf("network is required")
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
