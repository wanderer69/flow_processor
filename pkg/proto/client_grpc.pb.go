// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.21.12
// source: client.proto

package gen

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	ClientConnector_AddProcess_FullMethodName = "/processorclient.ClientConnector/AddProcess"
	ClientConnector_SetHandler_FullMethodName = "/processorclient.ClientConnector/SetHandler"
	ClientConnector_Connect_FullMethodName    = "/processorclient.ClientConnector/Connect"
)

// ClientConnectorClient is the client API for ClientConnector service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ClientConnectorClient interface {
	AddProcess(ctx context.Context, in *AddProcessRequest, opts ...grpc.CallOption) (*AddProcessResponse, error)
	SetHandler(ctx context.Context, in *SetHandlerRequest, opts ...grpc.CallOption) (*SetHandlerResponse, error)
	// rpc ConnectToProcess(ConnectToProcessRequest) returns (ConnectToProcessResponse) {}
	// rpc StartProcess(StartProcessRequest) returns (StartProcessResponse) {}
	Connect(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[Request, Response], error)
}

type clientConnectorClient struct {
	cc grpc.ClientConnInterface
}

func NewClientConnectorClient(cc grpc.ClientConnInterface) ClientConnectorClient {
	return &clientConnectorClient{cc}
}

func (c *clientConnectorClient) AddProcess(ctx context.Context, in *AddProcessRequest, opts ...grpc.CallOption) (*AddProcessResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(AddProcessResponse)
	err := c.cc.Invoke(ctx, ClientConnector_AddProcess_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clientConnectorClient) SetHandler(ctx context.Context, in *SetHandlerRequest, opts ...grpc.CallOption) (*SetHandlerResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SetHandlerResponse)
	err := c.cc.Invoke(ctx, ClientConnector_SetHandler_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clientConnectorClient) Connect(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[Request, Response], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &ClientConnector_ServiceDesc.Streams[0], ClientConnector_Connect_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[Request, Response]{ClientStream: stream}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type ClientConnector_ConnectClient = grpc.BidiStreamingClient[Request, Response]

// ClientConnectorServer is the server API for ClientConnector service.
// All implementations must embed UnimplementedClientConnectorServer
// for forward compatibility.
type ClientConnectorServer interface {
	AddProcess(context.Context, *AddProcessRequest) (*AddProcessResponse, error)
	SetHandler(context.Context, *SetHandlerRequest) (*SetHandlerResponse, error)
	// rpc ConnectToProcess(ConnectToProcessRequest) returns (ConnectToProcessResponse) {}
	// rpc StartProcess(StartProcessRequest) returns (StartProcessResponse) {}
	Connect(grpc.BidiStreamingServer[Request, Response]) error
	mustEmbedUnimplementedClientConnectorServer()
}

// UnimplementedClientConnectorServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedClientConnectorServer struct{}

func (UnimplementedClientConnectorServer) AddProcess(context.Context, *AddProcessRequest) (*AddProcessResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddProcess not implemented")
}
func (UnimplementedClientConnectorServer) SetHandler(context.Context, *SetHandlerRequest) (*SetHandlerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetHandler not implemented")
}
func (UnimplementedClientConnectorServer) Connect(grpc.BidiStreamingServer[Request, Response]) error {
	return status.Errorf(codes.Unimplemented, "method Connect not implemented")
}
func (UnimplementedClientConnectorServer) mustEmbedUnimplementedClientConnectorServer() {}
func (UnimplementedClientConnectorServer) testEmbeddedByValue()                         {}

// UnsafeClientConnectorServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ClientConnectorServer will
// result in compilation errors.
type UnsafeClientConnectorServer interface {
	mustEmbedUnimplementedClientConnectorServer()
}

func RegisterClientConnectorServer(s grpc.ServiceRegistrar, srv ClientConnectorServer) {
	// If the following call pancis, it indicates UnimplementedClientConnectorServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&ClientConnector_ServiceDesc, srv)
}

func _ClientConnector_AddProcess_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddProcessRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClientConnectorServer).AddProcess(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ClientConnector_AddProcess_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClientConnectorServer).AddProcess(ctx, req.(*AddProcessRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ClientConnector_SetHandler_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetHandlerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClientConnectorServer).SetHandler(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ClientConnector_SetHandler_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClientConnectorServer).SetHandler(ctx, req.(*SetHandlerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ClientConnector_Connect_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ClientConnectorServer).Connect(&grpc.GenericServerStream[Request, Response]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type ClientConnector_ConnectServer = grpc.BidiStreamingServer[Request, Response]

// ClientConnector_ServiceDesc is the grpc.ServiceDesc for ClientConnector service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ClientConnector_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "processorclient.ClientConnector",
	HandlerType: (*ClientConnectorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AddProcess",
			Handler:    _ClientConnector_AddProcess_Handler,
		},
		{
			MethodName: "SetHandler",
			Handler:    _ClientConnector_SetHandler_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Connect",
			Handler:       _ClientConnector_Connect_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "client.proto",
}