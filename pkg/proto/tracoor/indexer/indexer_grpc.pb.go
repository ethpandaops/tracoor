// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: indexer/indexer.proto

package indexer

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Indexer_GetStorageHandshakeToken_FullMethodName            = "/indexer.Indexer/GetStorageHandshakeToken"
	Indexer_CreateBeaconState_FullMethodName                   = "/indexer.Indexer/CreateBeaconState"
	Indexer_ListBeaconState_FullMethodName                     = "/indexer.Indexer/ListBeaconState"
	Indexer_CountBeaconState_FullMethodName                    = "/indexer.Indexer/CountBeaconState"
	Indexer_ListUniqueBeaconStateValues_FullMethodName         = "/indexer.Indexer/ListUniqueBeaconStateValues"
	Indexer_CreateExecutionBlockTrace_FullMethodName           = "/indexer.Indexer/CreateExecutionBlockTrace"
	Indexer_ListExecutionBlockTrace_FullMethodName             = "/indexer.Indexer/ListExecutionBlockTrace"
	Indexer_CountExecutionBlockTrace_FullMethodName            = "/indexer.Indexer/CountExecutionBlockTrace"
	Indexer_ListUniqueExecutionBlockTraceValues_FullMethodName = "/indexer.Indexer/ListUniqueExecutionBlockTraceValues"
	Indexer_CreateExecutionBadBlock_FullMethodName             = "/indexer.Indexer/CreateExecutionBadBlock"
	Indexer_ListExecutionBadBlock_FullMethodName               = "/indexer.Indexer/ListExecutionBadBlock"
	Indexer_CountExecutionBadBlock_FullMethodName              = "/indexer.Indexer/CountExecutionBadBlock"
	Indexer_ListUniqueExecutionBadBlockValues_FullMethodName   = "/indexer.Indexer/ListUniqueExecutionBadBlockValues"
)

// IndexerClient is the client API for Indexer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type IndexerClient interface {
	GetStorageHandshakeToken(ctx context.Context, in *GetStorageHandshakeTokenRequest, opts ...grpc.CallOption) (*GetStorageHandshakeTokenResponse, error)
	// BeaconState
	CreateBeaconState(ctx context.Context, in *CreateBeaconStateRequest, opts ...grpc.CallOption) (*CreateBeaconStateResponse, error)
	ListBeaconState(ctx context.Context, in *ListBeaconStateRequest, opts ...grpc.CallOption) (*ListBeaconStateResponse, error)
	CountBeaconState(ctx context.Context, in *CountBeaconStateRequest, opts ...grpc.CallOption) (*CountBeaconStateResponse, error)
	ListUniqueBeaconStateValues(ctx context.Context, in *ListUniqueBeaconStateValuesRequest, opts ...grpc.CallOption) (*ListUniqueBeaconStateValuesResponse, error)
	// ExecutionBlockTrace
	CreateExecutionBlockTrace(ctx context.Context, in *CreateExecutionBlockTraceRequest, opts ...grpc.CallOption) (*CreateExecutionBlockTraceResponse, error)
	ListExecutionBlockTrace(ctx context.Context, in *ListExecutionBlockTraceRequest, opts ...grpc.CallOption) (*ListExecutionBlockTraceResponse, error)
	CountExecutionBlockTrace(ctx context.Context, in *CountExecutionBlockTraceRequest, opts ...grpc.CallOption) (*CountExecutionBlockTraceResponse, error)
	ListUniqueExecutionBlockTraceValues(ctx context.Context, in *ListUniqueExecutionBlockTraceValuesRequest, opts ...grpc.CallOption) (*ListUniqueExecutionBlockTraceValuesResponse, error)
	// ExecutionBadBlock
	CreateExecutionBadBlock(ctx context.Context, in *CreateExecutionBadBlockRequest, opts ...grpc.CallOption) (*CreateExecutionBadBlockResponse, error)
	ListExecutionBadBlock(ctx context.Context, in *ListExecutionBadBlockRequest, opts ...grpc.CallOption) (*ListExecutionBadBlockResponse, error)
	CountExecutionBadBlock(ctx context.Context, in *CountExecutionBadBlockRequest, opts ...grpc.CallOption) (*CountExecutionBadBlockResponse, error)
	ListUniqueExecutionBadBlockValues(ctx context.Context, in *ListUniqueExecutionBadBlockValuesRequest, opts ...grpc.CallOption) (*ListUniqueExecutionBadBlockValuesResponse, error)
}

type indexerClient struct {
	cc grpc.ClientConnInterface
}

func NewIndexerClient(cc grpc.ClientConnInterface) IndexerClient {
	return &indexerClient{cc}
}

func (c *indexerClient) GetStorageHandshakeToken(ctx context.Context, in *GetStorageHandshakeTokenRequest, opts ...grpc.CallOption) (*GetStorageHandshakeTokenResponse, error) {
	out := new(GetStorageHandshakeTokenResponse)
	err := c.cc.Invoke(ctx, Indexer_GetStorageHandshakeToken_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *indexerClient) CreateBeaconState(ctx context.Context, in *CreateBeaconStateRequest, opts ...grpc.CallOption) (*CreateBeaconStateResponse, error) {
	out := new(CreateBeaconStateResponse)
	err := c.cc.Invoke(ctx, Indexer_CreateBeaconState_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *indexerClient) ListBeaconState(ctx context.Context, in *ListBeaconStateRequest, opts ...grpc.CallOption) (*ListBeaconStateResponse, error) {
	out := new(ListBeaconStateResponse)
	err := c.cc.Invoke(ctx, Indexer_ListBeaconState_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *indexerClient) CountBeaconState(ctx context.Context, in *CountBeaconStateRequest, opts ...grpc.CallOption) (*CountBeaconStateResponse, error) {
	out := new(CountBeaconStateResponse)
	err := c.cc.Invoke(ctx, Indexer_CountBeaconState_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *indexerClient) ListUniqueBeaconStateValues(ctx context.Context, in *ListUniqueBeaconStateValuesRequest, opts ...grpc.CallOption) (*ListUniqueBeaconStateValuesResponse, error) {
	out := new(ListUniqueBeaconStateValuesResponse)
	err := c.cc.Invoke(ctx, Indexer_ListUniqueBeaconStateValues_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *indexerClient) CreateExecutionBlockTrace(ctx context.Context, in *CreateExecutionBlockTraceRequest, opts ...grpc.CallOption) (*CreateExecutionBlockTraceResponse, error) {
	out := new(CreateExecutionBlockTraceResponse)
	err := c.cc.Invoke(ctx, Indexer_CreateExecutionBlockTrace_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *indexerClient) ListExecutionBlockTrace(ctx context.Context, in *ListExecutionBlockTraceRequest, opts ...grpc.CallOption) (*ListExecutionBlockTraceResponse, error) {
	out := new(ListExecutionBlockTraceResponse)
	err := c.cc.Invoke(ctx, Indexer_ListExecutionBlockTrace_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *indexerClient) CountExecutionBlockTrace(ctx context.Context, in *CountExecutionBlockTraceRequest, opts ...grpc.CallOption) (*CountExecutionBlockTraceResponse, error) {
	out := new(CountExecutionBlockTraceResponse)
	err := c.cc.Invoke(ctx, Indexer_CountExecutionBlockTrace_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *indexerClient) ListUniqueExecutionBlockTraceValues(ctx context.Context, in *ListUniqueExecutionBlockTraceValuesRequest, opts ...grpc.CallOption) (*ListUniqueExecutionBlockTraceValuesResponse, error) {
	out := new(ListUniqueExecutionBlockTraceValuesResponse)
	err := c.cc.Invoke(ctx, Indexer_ListUniqueExecutionBlockTraceValues_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *indexerClient) CreateExecutionBadBlock(ctx context.Context, in *CreateExecutionBadBlockRequest, opts ...grpc.CallOption) (*CreateExecutionBadBlockResponse, error) {
	out := new(CreateExecutionBadBlockResponse)
	err := c.cc.Invoke(ctx, Indexer_CreateExecutionBadBlock_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *indexerClient) ListExecutionBadBlock(ctx context.Context, in *ListExecutionBadBlockRequest, opts ...grpc.CallOption) (*ListExecutionBadBlockResponse, error) {
	out := new(ListExecutionBadBlockResponse)
	err := c.cc.Invoke(ctx, Indexer_ListExecutionBadBlock_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *indexerClient) CountExecutionBadBlock(ctx context.Context, in *CountExecutionBadBlockRequest, opts ...grpc.CallOption) (*CountExecutionBadBlockResponse, error) {
	out := new(CountExecutionBadBlockResponse)
	err := c.cc.Invoke(ctx, Indexer_CountExecutionBadBlock_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *indexerClient) ListUniqueExecutionBadBlockValues(ctx context.Context, in *ListUniqueExecutionBadBlockValuesRequest, opts ...grpc.CallOption) (*ListUniqueExecutionBadBlockValuesResponse, error) {
	out := new(ListUniqueExecutionBadBlockValuesResponse)
	err := c.cc.Invoke(ctx, Indexer_ListUniqueExecutionBadBlockValues_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// IndexerServer is the server API for Indexer service.
// All implementations must embed UnimplementedIndexerServer
// for forward compatibility
type IndexerServer interface {
	GetStorageHandshakeToken(context.Context, *GetStorageHandshakeTokenRequest) (*GetStorageHandshakeTokenResponse, error)
	// BeaconState
	CreateBeaconState(context.Context, *CreateBeaconStateRequest) (*CreateBeaconStateResponse, error)
	ListBeaconState(context.Context, *ListBeaconStateRequest) (*ListBeaconStateResponse, error)
	CountBeaconState(context.Context, *CountBeaconStateRequest) (*CountBeaconStateResponse, error)
	ListUniqueBeaconStateValues(context.Context, *ListUniqueBeaconStateValuesRequest) (*ListUniqueBeaconStateValuesResponse, error)
	// ExecutionBlockTrace
	CreateExecutionBlockTrace(context.Context, *CreateExecutionBlockTraceRequest) (*CreateExecutionBlockTraceResponse, error)
	ListExecutionBlockTrace(context.Context, *ListExecutionBlockTraceRequest) (*ListExecutionBlockTraceResponse, error)
	CountExecutionBlockTrace(context.Context, *CountExecutionBlockTraceRequest) (*CountExecutionBlockTraceResponse, error)
	ListUniqueExecutionBlockTraceValues(context.Context, *ListUniqueExecutionBlockTraceValuesRequest) (*ListUniqueExecutionBlockTraceValuesResponse, error)
	// ExecutionBadBlock
	CreateExecutionBadBlock(context.Context, *CreateExecutionBadBlockRequest) (*CreateExecutionBadBlockResponse, error)
	ListExecutionBadBlock(context.Context, *ListExecutionBadBlockRequest) (*ListExecutionBadBlockResponse, error)
	CountExecutionBadBlock(context.Context, *CountExecutionBadBlockRequest) (*CountExecutionBadBlockResponse, error)
	ListUniqueExecutionBadBlockValues(context.Context, *ListUniqueExecutionBadBlockValuesRequest) (*ListUniqueExecutionBadBlockValuesResponse, error)
	mustEmbedUnimplementedIndexerServer()
}

// UnimplementedIndexerServer must be embedded to have forward compatible implementations.
type UnimplementedIndexerServer struct {
}

func (UnimplementedIndexerServer) GetStorageHandshakeToken(context.Context, *GetStorageHandshakeTokenRequest) (*GetStorageHandshakeTokenResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetStorageHandshakeToken not implemented")
}
func (UnimplementedIndexerServer) CreateBeaconState(context.Context, *CreateBeaconStateRequest) (*CreateBeaconStateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateBeaconState not implemented")
}
func (UnimplementedIndexerServer) ListBeaconState(context.Context, *ListBeaconStateRequest) (*ListBeaconStateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListBeaconState not implemented")
}
func (UnimplementedIndexerServer) CountBeaconState(context.Context, *CountBeaconStateRequest) (*CountBeaconStateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CountBeaconState not implemented")
}
func (UnimplementedIndexerServer) ListUniqueBeaconStateValues(context.Context, *ListUniqueBeaconStateValuesRequest) (*ListUniqueBeaconStateValuesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListUniqueBeaconStateValues not implemented")
}
func (UnimplementedIndexerServer) CreateExecutionBlockTrace(context.Context, *CreateExecutionBlockTraceRequest) (*CreateExecutionBlockTraceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateExecutionBlockTrace not implemented")
}
func (UnimplementedIndexerServer) ListExecutionBlockTrace(context.Context, *ListExecutionBlockTraceRequest) (*ListExecutionBlockTraceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListExecutionBlockTrace not implemented")
}
func (UnimplementedIndexerServer) CountExecutionBlockTrace(context.Context, *CountExecutionBlockTraceRequest) (*CountExecutionBlockTraceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CountExecutionBlockTrace not implemented")
}
func (UnimplementedIndexerServer) ListUniqueExecutionBlockTraceValues(context.Context, *ListUniqueExecutionBlockTraceValuesRequest) (*ListUniqueExecutionBlockTraceValuesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListUniqueExecutionBlockTraceValues not implemented")
}
func (UnimplementedIndexerServer) CreateExecutionBadBlock(context.Context, *CreateExecutionBadBlockRequest) (*CreateExecutionBadBlockResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateExecutionBadBlock not implemented")
}
func (UnimplementedIndexerServer) ListExecutionBadBlock(context.Context, *ListExecutionBadBlockRequest) (*ListExecutionBadBlockResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListExecutionBadBlock not implemented")
}
func (UnimplementedIndexerServer) CountExecutionBadBlock(context.Context, *CountExecutionBadBlockRequest) (*CountExecutionBadBlockResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CountExecutionBadBlock not implemented")
}
func (UnimplementedIndexerServer) ListUniqueExecutionBadBlockValues(context.Context, *ListUniqueExecutionBadBlockValuesRequest) (*ListUniqueExecutionBadBlockValuesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListUniqueExecutionBadBlockValues not implemented")
}
func (UnimplementedIndexerServer) mustEmbedUnimplementedIndexerServer() {}

// UnsafeIndexerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to IndexerServer will
// result in compilation errors.
type UnsafeIndexerServer interface {
	mustEmbedUnimplementedIndexerServer()
}

func RegisterIndexerServer(s grpc.ServiceRegistrar, srv IndexerServer) {
	s.RegisterService(&Indexer_ServiceDesc, srv)
}

func _Indexer_GetStorageHandshakeToken_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetStorageHandshakeTokenRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServer).GetStorageHandshakeToken(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Indexer_GetStorageHandshakeToken_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServer).GetStorageHandshakeToken(ctx, req.(*GetStorageHandshakeTokenRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Indexer_CreateBeaconState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateBeaconStateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServer).CreateBeaconState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Indexer_CreateBeaconState_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServer).CreateBeaconState(ctx, req.(*CreateBeaconStateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Indexer_ListBeaconState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListBeaconStateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServer).ListBeaconState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Indexer_ListBeaconState_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServer).ListBeaconState(ctx, req.(*ListBeaconStateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Indexer_CountBeaconState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CountBeaconStateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServer).CountBeaconState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Indexer_CountBeaconState_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServer).CountBeaconState(ctx, req.(*CountBeaconStateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Indexer_ListUniqueBeaconStateValues_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListUniqueBeaconStateValuesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServer).ListUniqueBeaconStateValues(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Indexer_ListUniqueBeaconStateValues_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServer).ListUniqueBeaconStateValues(ctx, req.(*ListUniqueBeaconStateValuesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Indexer_CreateExecutionBlockTrace_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateExecutionBlockTraceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServer).CreateExecutionBlockTrace(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Indexer_CreateExecutionBlockTrace_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServer).CreateExecutionBlockTrace(ctx, req.(*CreateExecutionBlockTraceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Indexer_ListExecutionBlockTrace_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListExecutionBlockTraceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServer).ListExecutionBlockTrace(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Indexer_ListExecutionBlockTrace_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServer).ListExecutionBlockTrace(ctx, req.(*ListExecutionBlockTraceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Indexer_CountExecutionBlockTrace_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CountExecutionBlockTraceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServer).CountExecutionBlockTrace(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Indexer_CountExecutionBlockTrace_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServer).CountExecutionBlockTrace(ctx, req.(*CountExecutionBlockTraceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Indexer_ListUniqueExecutionBlockTraceValues_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListUniqueExecutionBlockTraceValuesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServer).ListUniqueExecutionBlockTraceValues(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Indexer_ListUniqueExecutionBlockTraceValues_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServer).ListUniqueExecutionBlockTraceValues(ctx, req.(*ListUniqueExecutionBlockTraceValuesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Indexer_CreateExecutionBadBlock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateExecutionBadBlockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServer).CreateExecutionBadBlock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Indexer_CreateExecutionBadBlock_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServer).CreateExecutionBadBlock(ctx, req.(*CreateExecutionBadBlockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Indexer_ListExecutionBadBlock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListExecutionBadBlockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServer).ListExecutionBadBlock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Indexer_ListExecutionBadBlock_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServer).ListExecutionBadBlock(ctx, req.(*ListExecutionBadBlockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Indexer_CountExecutionBadBlock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CountExecutionBadBlockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServer).CountExecutionBadBlock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Indexer_CountExecutionBadBlock_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServer).CountExecutionBadBlock(ctx, req.(*CountExecutionBadBlockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Indexer_ListUniqueExecutionBadBlockValues_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListUniqueExecutionBadBlockValuesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServer).ListUniqueExecutionBadBlockValues(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Indexer_ListUniqueExecutionBadBlockValues_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServer).ListUniqueExecutionBadBlockValues(ctx, req.(*ListUniqueExecutionBadBlockValuesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Indexer_ServiceDesc is the grpc.ServiceDesc for Indexer service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Indexer_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "indexer.Indexer",
	HandlerType: (*IndexerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetStorageHandshakeToken",
			Handler:    _Indexer_GetStorageHandshakeToken_Handler,
		},
		{
			MethodName: "CreateBeaconState",
			Handler:    _Indexer_CreateBeaconState_Handler,
		},
		{
			MethodName: "ListBeaconState",
			Handler:    _Indexer_ListBeaconState_Handler,
		},
		{
			MethodName: "CountBeaconState",
			Handler:    _Indexer_CountBeaconState_Handler,
		},
		{
			MethodName: "ListUniqueBeaconStateValues",
			Handler:    _Indexer_ListUniqueBeaconStateValues_Handler,
		},
		{
			MethodName: "CreateExecutionBlockTrace",
			Handler:    _Indexer_CreateExecutionBlockTrace_Handler,
		},
		{
			MethodName: "ListExecutionBlockTrace",
			Handler:    _Indexer_ListExecutionBlockTrace_Handler,
		},
		{
			MethodName: "CountExecutionBlockTrace",
			Handler:    _Indexer_CountExecutionBlockTrace_Handler,
		},
		{
			MethodName: "ListUniqueExecutionBlockTraceValues",
			Handler:    _Indexer_ListUniqueExecutionBlockTraceValues_Handler,
		},
		{
			MethodName: "CreateExecutionBadBlock",
			Handler:    _Indexer_CreateExecutionBadBlock_Handler,
		},
		{
			MethodName: "ListExecutionBadBlock",
			Handler:    _Indexer_ListExecutionBadBlock_Handler,
		},
		{
			MethodName: "CountExecutionBadBlock",
			Handler:    _Indexer_CountExecutionBadBlock_Handler,
		},
		{
			MethodName: "ListUniqueExecutionBadBlockValues",
			Handler:    _Indexer_ListUniqueExecutionBadBlockValues_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "indexer/indexer.proto",
}
