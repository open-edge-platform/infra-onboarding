// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             (unknown)
// source: internal/proto/workflow/v2/workflow.proto

package workflow

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

// WorkflowServiceClient is the client API for WorkflowService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type WorkflowServiceClient interface {
	// GetWorkflows creates a stream that will receive workflows intended for the agent identified
	// by the GetWorkflowsRequest.agent_id.
	GetWorkflows(ctx context.Context, in *GetWorkflowsRequest, opts ...grpc.CallOption) (WorkflowService_GetWorkflowsClient, error)
	// PublishEvent publishes a workflow event.
	PublishEvent(ctx context.Context, in *PublishEventRequest, opts ...grpc.CallOption) (*PublishEventResponse, error)
}

type workflowServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewWorkflowServiceClient(cc grpc.ClientConnInterface) WorkflowServiceClient {
	return &workflowServiceClient{cc}
}

func (c *workflowServiceClient) GetWorkflows(ctx context.Context, in *GetWorkflowsRequest, opts ...grpc.CallOption) (WorkflowService_GetWorkflowsClient, error) {
	stream, err := c.cc.NewStream(ctx, &WorkflowService_ServiceDesc.Streams[0], "/internal.proto.workflow.v2.WorkflowService/GetWorkflows", opts...)
	if err != nil {
		return nil, err
	}
	x := &workflowServiceGetWorkflowsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type WorkflowService_GetWorkflowsClient interface {
	Recv() (*GetWorkflowsResponse, error)
	grpc.ClientStream
}

type workflowServiceGetWorkflowsClient struct {
	grpc.ClientStream
}

func (x *workflowServiceGetWorkflowsClient) Recv() (*GetWorkflowsResponse, error) {
	m := new(GetWorkflowsResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *workflowServiceClient) PublishEvent(ctx context.Context, in *PublishEventRequest, opts ...grpc.CallOption) (*PublishEventResponse, error) {
	out := new(PublishEventResponse)
	err := c.cc.Invoke(ctx, "/internal.proto.workflow.v2.WorkflowService/PublishEvent", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// WorkflowServiceServer is the server API for WorkflowService service.
// All implementations should embed UnimplementedWorkflowServiceServer
// for forward compatibility
type WorkflowServiceServer interface {
	// GetWorkflows creates a stream that will receive workflows intended for the agent identified
	// by the GetWorkflowsRequest.agent_id.
	GetWorkflows(*GetWorkflowsRequest, WorkflowService_GetWorkflowsServer) error
	// PublishEvent publishes a workflow event.
	PublishEvent(context.Context, *PublishEventRequest) (*PublishEventResponse, error)
}

// UnimplementedWorkflowServiceServer should be embedded to have forward compatible implementations.
type UnimplementedWorkflowServiceServer struct{}

func (UnimplementedWorkflowServiceServer) GetWorkflows(*GetWorkflowsRequest, WorkflowService_GetWorkflowsServer) error {
	return status.Errorf(codes.Unimplemented, "method GetWorkflows not implemented")
}

func (UnimplementedWorkflowServiceServer) PublishEvent(context.Context, *PublishEventRequest) (*PublishEventResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PublishEvent not implemented")
}

// UnsafeWorkflowServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to WorkflowServiceServer will
// result in compilation errors.
type UnsafeWorkflowServiceServer interface {
	mustEmbedUnimplementedWorkflowServiceServer()
}

func RegisterWorkflowServiceServer(s grpc.ServiceRegistrar, srv WorkflowServiceServer) {
	s.RegisterService(&WorkflowService_ServiceDesc, srv)
}

func _WorkflowService_GetWorkflows_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetWorkflowsRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(WorkflowServiceServer).GetWorkflows(m, &workflowServiceGetWorkflowsServer{stream})
}

type WorkflowService_GetWorkflowsServer interface {
	Send(*GetWorkflowsResponse) error
	grpc.ServerStream
}

type workflowServiceGetWorkflowsServer struct {
	grpc.ServerStream
}

func (x *workflowServiceGetWorkflowsServer) Send(m *GetWorkflowsResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _WorkflowService_PublishEvent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PublishEventRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkflowServiceServer).PublishEvent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/internal.proto.workflow.v2.WorkflowService/PublishEvent",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkflowServiceServer).PublishEvent(ctx, req.(*PublishEventRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// WorkflowService_ServiceDesc is the grpc.ServiceDesc for WorkflowService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var WorkflowService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "internal.proto.workflow.v2.WorkflowService",
	HandlerType: (*WorkflowServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PublishEvent",
			Handler:    _WorkflowService_PublishEvent_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetWorkflows",
			Handler:       _WorkflowService_GetWorkflows_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "internal/proto/workflow/v2/workflow.proto",
}
