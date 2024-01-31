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
	Indexer_CreateBeaconState_FullMethodName = "/indexer.Indexer/CreateBeaconState"
	Indexer_ListBeaconState_FullMethodName   = "/indexer.Indexer/ListBeaconState"
	Indexer_ListUniqueValues_FullMethodName  = "/indexer.Indexer/ListUniqueValues"
)

// IndexerClient is the client API for Indexer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type IndexerClient interface {
	// BeaconState
	CreateBeaconState(ctx context.Context, in *CreateBeaconStateRequest, opts ...grpc.CallOption) (*CreateBeaconStateResponse, error)
	ListBeaconState(ctx context.Context, in *ListBeaconStateRequest, opts ...grpc.CallOption) (*ListBeaconStateResponse, error)
	ListUniqueValues(ctx context.Context, in *ListUniqueValuesRequest, opts ...grpc.CallOption) (*ListUniqueValuesResponse, error)
}

type indexerClient struct {
	cc grpc.ClientConnInterface
}

func NewIndexerClient(cc grpc.ClientConnInterface) IndexerClient {
	return &indexerClient{cc}
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

func (c *indexerClient) ListUniqueValues(ctx context.Context, in *ListUniqueValuesRequest, opts ...grpc.CallOption) (*ListUniqueValuesResponse, error) {
	out := new(ListUniqueValuesResponse)
	err := c.cc.Invoke(ctx, Indexer_ListUniqueValues_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// IndexerServer is the server API for Indexer service.
// All implementations must embed UnimplementedIndexerServer
// for forward compatibility
type IndexerServer interface {
	// BeaconState
	CreateBeaconState(context.Context, *CreateBeaconStateRequest) (*CreateBeaconStateResponse, error)
	ListBeaconState(context.Context, *ListBeaconStateRequest) (*ListBeaconStateResponse, error)
	ListUniqueValues(context.Context, *ListUniqueValuesRequest) (*ListUniqueValuesResponse, error)
	mustEmbedUnimplementedIndexerServer()
}

// UnimplementedIndexerServer must be embedded to have forward compatible implementations.
type UnimplementedIndexerServer struct {
}

func (UnimplementedIndexerServer) CreateBeaconState(context.Context, *CreateBeaconStateRequest) (*CreateBeaconStateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateBeaconState not implemented")
}
func (UnimplementedIndexerServer) ListBeaconState(context.Context, *ListBeaconStateRequest) (*ListBeaconStateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListBeaconState not implemented")
}
func (UnimplementedIndexerServer) ListUniqueValues(context.Context, *ListUniqueValuesRequest) (*ListUniqueValuesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListUniqueValues not implemented")
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

func _Indexer_ListUniqueValues_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListUniqueValuesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServer).ListUniqueValues(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Indexer_ListUniqueValues_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServer).ListUniqueValues(ctx, req.(*ListUniqueValuesRequest))
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
			MethodName: "CreateBeaconState",
			Handler:    _Indexer_CreateBeaconState_Handler,
		},
		{
			MethodName: "ListBeaconState",
			Handler:    _Indexer_ListBeaconState_Handler,
		},
		{
			MethodName: "ListUniqueValues",
			Handler:    _Indexer_ListUniqueValues_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "indexer/indexer.proto",
}
