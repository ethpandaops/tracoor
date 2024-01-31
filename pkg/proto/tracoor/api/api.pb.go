// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        (unknown)
// source: api/api.proto

package api

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ListUniqueBeaconStateValuesRequest_Field int32

const (
	ListUniqueBeaconStateValuesRequest_node         ListUniqueBeaconStateValuesRequest_Field = 0
	ListUniqueBeaconStateValuesRequest_slot         ListUniqueBeaconStateValuesRequest_Field = 1
	ListUniqueBeaconStateValuesRequest_epoch        ListUniqueBeaconStateValuesRequest_Field = 2
	ListUniqueBeaconStateValuesRequest_state_root   ListUniqueBeaconStateValuesRequest_Field = 3
	ListUniqueBeaconStateValuesRequest_node_version ListUniqueBeaconStateValuesRequest_Field = 4
	ListUniqueBeaconStateValuesRequest_network      ListUniqueBeaconStateValuesRequest_Field = 5
)

// Enum value maps for ListUniqueBeaconStateValuesRequest_Field.
var (
	ListUniqueBeaconStateValuesRequest_Field_name = map[int32]string{
		0: "node",
		1: "slot",
		2: "epoch",
		3: "state_root",
		4: "node_version",
		5: "network",
	}
	ListUniqueBeaconStateValuesRequest_Field_value = map[string]int32{
		"node":         0,
		"slot":         1,
		"epoch":        2,
		"state_root":   3,
		"node_version": 4,
		"network":      5,
	}
)

func (x ListUniqueBeaconStateValuesRequest_Field) Enum() *ListUniqueBeaconStateValuesRequest_Field {
	p := new(ListUniqueBeaconStateValuesRequest_Field)
	*p = x
	return p
}

func (x ListUniqueBeaconStateValuesRequest_Field) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ListUniqueBeaconStateValuesRequest_Field) Descriptor() protoreflect.EnumDescriptor {
	return file_api_api_proto_enumTypes[0].Descriptor()
}

func (ListUniqueBeaconStateValuesRequest_Field) Type() protoreflect.EnumType {
	return &file_api_api_proto_enumTypes[0]
}

func (x ListUniqueBeaconStateValuesRequest_Field) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ListUniqueBeaconStateValuesRequest_Field.Descriptor instead.
func (ListUniqueBeaconStateValuesRequest_Field) EnumDescriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{3, 0}
}

type BeaconState struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id          *wrapperspb.StringValue `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Node        *wrapperspb.StringValue `protobuf:"bytes,2,opt,name=node,proto3" json:"node,omitempty"`
	FetchedAt   *timestamppb.Timestamp  `protobuf:"bytes,3,opt,name=fetched_at,proto3" json:"fetched_at,omitempty"`
	Slot        *wrapperspb.UInt64Value `protobuf:"bytes,4,opt,name=slot,proto3" json:"slot,omitempty"`
	Epoch       *wrapperspb.UInt64Value `protobuf:"bytes,5,opt,name=epoch,proto3" json:"epoch,omitempty"`
	StateRoot   *wrapperspb.StringValue `protobuf:"bytes,6,opt,name=state_root,proto3" json:"state_root,omitempty"`
	NodeVersion *wrapperspb.StringValue `protobuf:"bytes,7,opt,name=node_version,proto3" json:"node_version,omitempty"`
	Network     *wrapperspb.StringValue `protobuf:"bytes,8,opt,name=network,proto3" json:"network,omitempty"`
}

func (x *BeaconState) Reset() {
	*x = BeaconState{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BeaconState) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BeaconState) ProtoMessage() {}

func (x *BeaconState) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BeaconState.ProtoReflect.Descriptor instead.
func (*BeaconState) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{0}
}

func (x *BeaconState) GetId() *wrapperspb.StringValue {
	if x != nil {
		return x.Id
	}
	return nil
}

func (x *BeaconState) GetNode() *wrapperspb.StringValue {
	if x != nil {
		return x.Node
	}
	return nil
}

func (x *BeaconState) GetFetchedAt() *timestamppb.Timestamp {
	if x != nil {
		return x.FetchedAt
	}
	return nil
}

func (x *BeaconState) GetSlot() *wrapperspb.UInt64Value {
	if x != nil {
		return x.Slot
	}
	return nil
}

func (x *BeaconState) GetEpoch() *wrapperspb.UInt64Value {
	if x != nil {
		return x.Epoch
	}
	return nil
}

func (x *BeaconState) GetStateRoot() *wrapperspb.StringValue {
	if x != nil {
		return x.StateRoot
	}
	return nil
}

func (x *BeaconState) GetNodeVersion() *wrapperspb.StringValue {
	if x != nil {
		return x.NodeVersion
	}
	return nil
}

func (x *BeaconState) GetNetwork() *wrapperspb.StringValue {
	if x != nil {
		return x.Network
	}
	return nil
}

type ListBeaconStateRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Node        string                 `protobuf:"bytes,1,opt,name=node,proto3" json:"node,omitempty"`
	Slot        uint64                 `protobuf:"varint,2,opt,name=slot,proto3" json:"slot,omitempty"`
	Epoch       uint64                 `protobuf:"varint,3,opt,name=epoch,proto3" json:"epoch,omitempty"`
	StateRoot   string                 `protobuf:"bytes,4,opt,name=state_root,proto3" json:"state_root,omitempty"`
	NodeVersion string                 `protobuf:"bytes,5,opt,name=node_version,proto3" json:"node_version,omitempty"`
	Network     string                 `protobuf:"bytes,6,opt,name=network,proto3" json:"network,omitempty"`
	Before      *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=before,proto3" json:"before,omitempty"`
	After       *timestamppb.Timestamp `protobuf:"bytes,8,opt,name=after,proto3" json:"after,omitempty"`
	Pagination  *PaginationCursor      `protobuf:"bytes,9,opt,name=pagination,proto3" json:"pagination,omitempty"`
}

func (x *ListBeaconStateRequest) Reset() {
	*x = ListBeaconStateRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListBeaconStateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListBeaconStateRequest) ProtoMessage() {}

func (x *ListBeaconStateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListBeaconStateRequest.ProtoReflect.Descriptor instead.
func (*ListBeaconStateRequest) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{1}
}

func (x *ListBeaconStateRequest) GetNode() string {
	if x != nil {
		return x.Node
	}
	return ""
}

func (x *ListBeaconStateRequest) GetSlot() uint64 {
	if x != nil {
		return x.Slot
	}
	return 0
}

func (x *ListBeaconStateRequest) GetEpoch() uint64 {
	if x != nil {
		return x.Epoch
	}
	return 0
}

func (x *ListBeaconStateRequest) GetStateRoot() string {
	if x != nil {
		return x.StateRoot
	}
	return ""
}

func (x *ListBeaconStateRequest) GetNodeVersion() string {
	if x != nil {
		return x.NodeVersion
	}
	return ""
}

func (x *ListBeaconStateRequest) GetNetwork() string {
	if x != nil {
		return x.Network
	}
	return ""
}

func (x *ListBeaconStateRequest) GetBefore() *timestamppb.Timestamp {
	if x != nil {
		return x.Before
	}
	return nil
}

func (x *ListBeaconStateRequest) GetAfter() *timestamppb.Timestamp {
	if x != nil {
		return x.After
	}
	return nil
}

func (x *ListBeaconStateRequest) GetPagination() *PaginationCursor {
	if x != nil {
		return x.Pagination
	}
	return nil
}

type ListBeaconStateResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BeaconStates []*BeaconState `protobuf:"bytes,1,rep,name=beacon_states,proto3" json:"beacon_states,omitempty"`
}

func (x *ListBeaconStateResponse) Reset() {
	*x = ListBeaconStateResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListBeaconStateResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListBeaconStateResponse) ProtoMessage() {}

func (x *ListBeaconStateResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListBeaconStateResponse.ProtoReflect.Descriptor instead.
func (*ListBeaconStateResponse) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{2}
}

func (x *ListBeaconStateResponse) GetBeaconStates() []*BeaconState {
	if x != nil {
		return x.BeaconStates
	}
	return nil
}

type ListUniqueBeaconStateValuesRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Fields []ListUniqueBeaconStateValuesRequest_Field `protobuf:"varint,1,rep,packed,name=fields,proto3,enum=api.ListUniqueBeaconStateValuesRequest_Field" json:"fields,omitempty"`
}

func (x *ListUniqueBeaconStateValuesRequest) Reset() {
	*x = ListUniqueBeaconStateValuesRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListUniqueBeaconStateValuesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListUniqueBeaconStateValuesRequest) ProtoMessage() {}

func (x *ListUniqueBeaconStateValuesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListUniqueBeaconStateValuesRequest.ProtoReflect.Descriptor instead.
func (*ListUniqueBeaconStateValuesRequest) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{3}
}

func (x *ListUniqueBeaconStateValuesRequest) GetFields() []ListUniqueBeaconStateValuesRequest_Field {
	if x != nil {
		return x.Fields
	}
	return nil
}

type ListUniqueBeaconStateValuesResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Node        []string `protobuf:"bytes,1,rep,name=node,proto3" json:"node,omitempty"`
	Slot        []uint64 `protobuf:"varint,2,rep,packed,name=slot,proto3" json:"slot,omitempty"`
	Epoch       []uint64 `protobuf:"varint,3,rep,packed,name=epoch,proto3" json:"epoch,omitempty"`
	StateRoot   []string `protobuf:"bytes,4,rep,name=state_root,proto3" json:"state_root,omitempty"`
	NodeVersion []string `protobuf:"bytes,5,rep,name=node_version,proto3" json:"node_version,omitempty"`
	Network     []string `protobuf:"bytes,6,rep,name=network,proto3" json:"network,omitempty"`
}

func (x *ListUniqueBeaconStateValuesResponse) Reset() {
	*x = ListUniqueBeaconStateValuesResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListUniqueBeaconStateValuesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListUniqueBeaconStateValuesResponse) ProtoMessage() {}

func (x *ListUniqueBeaconStateValuesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListUniqueBeaconStateValuesResponse.ProtoReflect.Descriptor instead.
func (*ListUniqueBeaconStateValuesResponse) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{4}
}

func (x *ListUniqueBeaconStateValuesResponse) GetNode() []string {
	if x != nil {
		return x.Node
	}
	return nil
}

func (x *ListUniqueBeaconStateValuesResponse) GetSlot() []uint64 {
	if x != nil {
		return x.Slot
	}
	return nil
}

func (x *ListUniqueBeaconStateValuesResponse) GetEpoch() []uint64 {
	if x != nil {
		return x.Epoch
	}
	return nil
}

func (x *ListUniqueBeaconStateValuesResponse) GetStateRoot() []string {
	if x != nil {
		return x.StateRoot
	}
	return nil
}

func (x *ListUniqueBeaconStateValuesResponse) GetNodeVersion() []string {
	if x != nil {
		return x.NodeVersion
	}
	return nil
}

func (x *ListUniqueBeaconStateValuesResponse) GetNetwork() []string {
	if x != nil {
		return x.Network
	}
	return nil
}

type PaginationCursor struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Limit   int32  `protobuf:"varint,1,opt,name=limit,proto3" json:"limit,omitempty"`
	Offset  int32  `protobuf:"varint,2,opt,name=offset,proto3" json:"offset,omitempty"`
	OrderBy string `protobuf:"bytes,3,opt,name=order_by,json=orderBy,proto3" json:"order_by,omitempty"`
}

func (x *PaginationCursor) Reset() {
	*x = PaginationCursor{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PaginationCursor) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PaginationCursor) ProtoMessage() {}

func (x *PaginationCursor) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PaginationCursor.ProtoReflect.Descriptor instead.
func (*PaginationCursor) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{5}
}

func (x *PaginationCursor) GetLimit() int32 {
	if x != nil {
		return x.Limit
	}
	return 0
}

func (x *PaginationCursor) GetOffset() int32 {
	if x != nil {
		return x.Offset
	}
	return 0
}

func (x *PaginationCursor) GetOrderBy() string {
	if x != nil {
		return x.OrderBy
	}
	return ""
}

var File_api_api_proto protoreflect.FileDescriptor

var file_api_api_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x70, 0x69, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x03, 0x61, 0x70, 0x69, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x77, 0x72, 0x61, 0x70, 0x70, 0x65, 0x72, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xc7, 0x03, 0x0a, 0x0b, 0x42, 0x65, 0x61, 0x63, 0x6f, 0x6e,
	0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x2c, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52,
	0x02, 0x69, 0x64, 0x12, 0x30, 0x0a, 0x04, 0x6e, 0x6f, 0x64, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52,
	0x04, 0x6e, 0x6f, 0x64, 0x65, 0x12, 0x3a, 0x0a, 0x0a, 0x66, 0x65, 0x74, 0x63, 0x68, 0x65, 0x64,
	0x5f, 0x61, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65,
	0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0a, 0x66, 0x65, 0x74, 0x63, 0x68, 0x65, 0x64, 0x5f, 0x61,
	0x74, 0x12, 0x30, 0x0a, 0x04, 0x73, 0x6c, 0x6f, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x04, 0x73,
	0x6c, 0x6f, 0x74, 0x12, 0x32, 0x0a, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61, 0x6c, 0x75, 0x65,
	0x52, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x12, 0x3c, 0x0a, 0x0a, 0x73, 0x74, 0x61, 0x74, 0x65,
	0x5f, 0x72, 0x6f, 0x6f, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74,
	0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x0a, 0x73, 0x74, 0x61, 0x74, 0x65,
	0x5f, 0x72, 0x6f, 0x6f, 0x74, 0x12, 0x40, 0x0a, 0x0c, 0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x76, 0x65,
	0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74,
	0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x0c, 0x6e, 0x6f, 0x64, 0x65, 0x5f,
	0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x36, 0x0a, 0x07, 0x6e, 0x65, 0x74, 0x77, 0x6f,
	0x72, 0x6b, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e,
	0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x07, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x22,
	0xd1, 0x02, 0x0a, 0x16, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x65, 0x61, 0x63, 0x6f, 0x6e, 0x53, 0x74,
	0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x6f,
	0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x6f, 0x64, 0x65, 0x12, 0x12,
	0x0a, 0x04, 0x73, 0x6c, 0x6f, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x04, 0x73, 0x6c,
	0x6f, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x04, 0x52, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x12, 0x1e, 0x0a, 0x0a, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x5f, 0x72, 0x6f, 0x6f, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x73, 0x74,
	0x61, 0x74, 0x65, 0x5f, 0x72, 0x6f, 0x6f, 0x74, 0x12, 0x22, 0x0a, 0x0c, 0x6e, 0x6f, 0x64, 0x65,
	0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c,
	0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x18, 0x0a, 0x07,
	0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6e,
	0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x12, 0x32, 0x0a, 0x06, 0x62, 0x65, 0x66, 0x6f, 0x72, 0x65,
	0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61,
	0x6d, 0x70, 0x52, 0x06, 0x62, 0x65, 0x66, 0x6f, 0x72, 0x65, 0x12, 0x30, 0x0a, 0x05, 0x61, 0x66,
	0x74, 0x65, 0x72, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65,
	0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x05, 0x61, 0x66, 0x74, 0x65, 0x72, 0x12, 0x35, 0x0a, 0x0a,
	0x70, 0x61, 0x67, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x15, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x50, 0x61, 0x67, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x43, 0x75, 0x72, 0x73, 0x6f, 0x72, 0x52, 0x0a, 0x70, 0x61, 0x67, 0x69, 0x6e, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x22, 0x51, 0x0a, 0x17, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x65, 0x61, 0x63, 0x6f,
	0x6e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x36,
	0x0a, 0x0d, 0x62, 0x65, 0x61, 0x63, 0x6f, 0x6e, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x65, 0x73, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x42, 0x65, 0x61, 0x63,
	0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x0d, 0x62, 0x65, 0x61, 0x63, 0x6f, 0x6e, 0x5f,
	0x73, 0x74, 0x61, 0x74, 0x65, 0x73, 0x22, 0xc2, 0x01, 0x0a, 0x22, 0x4c, 0x69, 0x73, 0x74, 0x55,
	0x6e, 0x69, 0x71, 0x75, 0x65, 0x42, 0x65, 0x61, 0x63, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x65,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x45, 0x0a,
	0x06, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0e, 0x32, 0x2d, 0x2e,
	0x61, 0x70, 0x69, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x55, 0x6e, 0x69, 0x71, 0x75, 0x65, 0x42, 0x65,
	0x61, 0x63, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x52, 0x06, 0x66, 0x69,
	0x65, 0x6c, 0x64, 0x73, 0x22, 0x55, 0x0a, 0x05, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x08, 0x0a,
	0x04, 0x6e, 0x6f, 0x64, 0x65, 0x10, 0x00, 0x12, 0x08, 0x0a, 0x04, 0x73, 0x6c, 0x6f, 0x74, 0x10,
	0x01, 0x12, 0x09, 0x0a, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x10, 0x02, 0x12, 0x0e, 0x0a, 0x0a,
	0x73, 0x74, 0x61, 0x74, 0x65, 0x5f, 0x72, 0x6f, 0x6f, 0x74, 0x10, 0x03, 0x12, 0x10, 0x0a, 0x0c,
	0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x10, 0x04, 0x12, 0x0b,
	0x0a, 0x07, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x10, 0x05, 0x22, 0xc1, 0x01, 0x0a, 0x23,
	0x4c, 0x69, 0x73, 0x74, 0x55, 0x6e, 0x69, 0x71, 0x75, 0x65, 0x42, 0x65, 0x61, 0x63, 0x6f, 0x6e,
	0x53, 0x74, 0x61, 0x74, 0x65, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x6f, 0x64, 0x65, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x09, 0x52, 0x04, 0x6e, 0x6f, 0x64, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x6c, 0x6f, 0x74, 0x18,
	0x02, 0x20, 0x03, 0x28, 0x04, 0x52, 0x04, 0x73, 0x6c, 0x6f, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x65,
	0x70, 0x6f, 0x63, 0x68, 0x18, 0x03, 0x20, 0x03, 0x28, 0x04, 0x52, 0x05, 0x65, 0x70, 0x6f, 0x63,
	0x68, 0x12, 0x1e, 0x0a, 0x0a, 0x73, 0x74, 0x61, 0x74, 0x65, 0x5f, 0x72, 0x6f, 0x6f, 0x74, 0x18,
	0x04, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0a, 0x73, 0x74, 0x61, 0x74, 0x65, 0x5f, 0x72, 0x6f, 0x6f,
	0x74, 0x12, 0x22, 0x0a, 0x0c, 0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f,
	0x6e, 0x18, 0x05, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0c, 0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x76, 0x65,
	0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x18, 0x0a, 0x07, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b,
	0x18, 0x06, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x22,
	0x5b, 0x0a, 0x10, 0x50, 0x61, 0x67, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x75, 0x72,
	0x73, 0x6f, 0x72, 0x12, 0x14, 0x0a, 0x05, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x05, 0x52, 0x05, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x6f, 0x66, 0x66,
	0x73, 0x65, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65,
	0x74, 0x12, 0x19, 0x0a, 0x08, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x5f, 0x62, 0x79, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x07, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x42, 0x79, 0x32, 0xc9, 0x01, 0x0a,
	0x03, 0x41, 0x50, 0x49, 0x12, 0x4e, 0x0a, 0x0f, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x65, 0x61, 0x63,
	0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x1b, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x4c, 0x69,
	0x73, 0x74, 0x42, 0x65, 0x61, 0x63, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x1c, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x42,
	0x65, 0x61, 0x63, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x22, 0x00, 0x12, 0x72, 0x0a, 0x1b, 0x4c, 0x69, 0x73, 0x74, 0x55, 0x6e, 0x69, 0x71,
	0x75, 0x65, 0x42, 0x65, 0x61, 0x63, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x56, 0x61, 0x6c,
	0x75, 0x65, 0x73, 0x12, 0x27, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x55, 0x6e,
	0x69, 0x71, 0x75, 0x65, 0x42, 0x65, 0x61, 0x63, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x28, 0x2e, 0x61,
	0x70, 0x69, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x55, 0x6e, 0x69, 0x71, 0x75, 0x65, 0x42, 0x65, 0x61,
	0x63, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x36, 0x5a, 0x34, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x74, 0x68, 0x70, 0x61, 0x6e, 0x64, 0x61, 0x6f,
	0x70, 0x73, 0x2f, 0x74, 0x72, 0x61, 0x63, 0x6f, 0x6f, 0x72, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x74, 0x72, 0x61, 0x63, 0x6f, 0x6f, 0x72, 0x2f, 0x61, 0x70, 0x69,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_api_api_proto_rawDescOnce sync.Once
	file_api_api_proto_rawDescData = file_api_api_proto_rawDesc
)

func file_api_api_proto_rawDescGZIP() []byte {
	file_api_api_proto_rawDescOnce.Do(func() {
		file_api_api_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_api_proto_rawDescData)
	})
	return file_api_api_proto_rawDescData
}

var file_api_api_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_api_api_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_api_api_proto_goTypes = []interface{}{
	(ListUniqueBeaconStateValuesRequest_Field)(0), // 0: api.ListUniqueBeaconStateValuesRequest.Field
	(*BeaconState)(nil),                           // 1: api.BeaconState
	(*ListBeaconStateRequest)(nil),                // 2: api.ListBeaconStateRequest
	(*ListBeaconStateResponse)(nil),               // 3: api.ListBeaconStateResponse
	(*ListUniqueBeaconStateValuesRequest)(nil),    // 4: api.ListUniqueBeaconStateValuesRequest
	(*ListUniqueBeaconStateValuesResponse)(nil),   // 5: api.ListUniqueBeaconStateValuesResponse
	(*PaginationCursor)(nil),                      // 6: api.PaginationCursor
	(*wrapperspb.StringValue)(nil),                // 7: google.protobuf.StringValue
	(*timestamppb.Timestamp)(nil),                 // 8: google.protobuf.Timestamp
	(*wrapperspb.UInt64Value)(nil),                // 9: google.protobuf.UInt64Value
}
var file_api_api_proto_depIdxs = []int32{
	7,  // 0: api.BeaconState.id:type_name -> google.protobuf.StringValue
	7,  // 1: api.BeaconState.node:type_name -> google.protobuf.StringValue
	8,  // 2: api.BeaconState.fetched_at:type_name -> google.protobuf.Timestamp
	9,  // 3: api.BeaconState.slot:type_name -> google.protobuf.UInt64Value
	9,  // 4: api.BeaconState.epoch:type_name -> google.protobuf.UInt64Value
	7,  // 5: api.BeaconState.state_root:type_name -> google.protobuf.StringValue
	7,  // 6: api.BeaconState.node_version:type_name -> google.protobuf.StringValue
	7,  // 7: api.BeaconState.network:type_name -> google.protobuf.StringValue
	8,  // 8: api.ListBeaconStateRequest.before:type_name -> google.protobuf.Timestamp
	8,  // 9: api.ListBeaconStateRequest.after:type_name -> google.protobuf.Timestamp
	6,  // 10: api.ListBeaconStateRequest.pagination:type_name -> api.PaginationCursor
	1,  // 11: api.ListBeaconStateResponse.beacon_states:type_name -> api.BeaconState
	0,  // 12: api.ListUniqueBeaconStateValuesRequest.fields:type_name -> api.ListUniqueBeaconStateValuesRequest.Field
	2,  // 13: api.API.ListBeaconState:input_type -> api.ListBeaconStateRequest
	4,  // 14: api.API.ListUniqueBeaconStateValues:input_type -> api.ListUniqueBeaconStateValuesRequest
	3,  // 15: api.API.ListBeaconState:output_type -> api.ListBeaconStateResponse
	5,  // 16: api.API.ListUniqueBeaconStateValues:output_type -> api.ListUniqueBeaconStateValuesResponse
	15, // [15:17] is the sub-list for method output_type
	13, // [13:15] is the sub-list for method input_type
	13, // [13:13] is the sub-list for extension type_name
	13, // [13:13] is the sub-list for extension extendee
	0,  // [0:13] is the sub-list for field type_name
}

func init() { file_api_api_proto_init() }
func file_api_api_proto_init() {
	if File_api_api_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_api_api_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BeaconState); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_api_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListBeaconStateRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_api_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListBeaconStateResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_api_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListUniqueBeaconStateValuesRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_api_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListUniqueBeaconStateValuesResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_api_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PaginationCursor); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_api_api_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_api_api_proto_goTypes,
		DependencyIndexes: file_api_api_proto_depIdxs,
		EnumInfos:         file_api_api_proto_enumTypes,
		MessageInfos:      file_api_api_proto_msgTypes,
	}.Build()
	File_api_api_proto = out.File
	file_api_api_proto_rawDesc = nil
	file_api_api_proto_goTypes = nil
	file_api_api_proto_depIdxs = nil
}
