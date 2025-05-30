// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: agent_working_schedule.proto

package wfm

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
	AgentWorkingScheduleService_CreateAgentsWorkingScheduleShifts_FullMethodName = "/wfm.AgentWorkingScheduleService/CreateAgentsWorkingScheduleShifts"
	AgentWorkingScheduleService_SearchAgentsWorkingSchedule_FullMethodName       = "/wfm.AgentWorkingScheduleService/SearchAgentsWorkingSchedule"
)

// AgentWorkingScheduleServiceClient is the client API for AgentWorkingScheduleService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AgentWorkingScheduleServiceClient interface {
	CreateAgentsWorkingScheduleShifts(ctx context.Context, in *CreateAgentsWorkingScheduleShiftsRequest, opts ...grpc.CallOption) (*CreateAgentsWorkingScheduleShiftsResponse, error)
	SearchAgentsWorkingSchedule(ctx context.Context, in *SearchAgentsWorkingScheduleRequest, opts ...grpc.CallOption) (*SearchAgentsWorkingScheduleResponse, error)
}

type agentWorkingScheduleServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewAgentWorkingScheduleServiceClient(cc grpc.ClientConnInterface) AgentWorkingScheduleServiceClient {
	return &agentWorkingScheduleServiceClient{cc}
}

func (c *agentWorkingScheduleServiceClient) CreateAgentsWorkingScheduleShifts(ctx context.Context, in *CreateAgentsWorkingScheduleShiftsRequest, opts ...grpc.CallOption) (*CreateAgentsWorkingScheduleShiftsResponse, error) {
	out := new(CreateAgentsWorkingScheduleShiftsResponse)
	err := c.cc.Invoke(ctx, AgentWorkingScheduleService_CreateAgentsWorkingScheduleShifts_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentWorkingScheduleServiceClient) SearchAgentsWorkingSchedule(ctx context.Context, in *SearchAgentsWorkingScheduleRequest, opts ...grpc.CallOption) (*SearchAgentsWorkingScheduleResponse, error) {
	out := new(SearchAgentsWorkingScheduleResponse)
	err := c.cc.Invoke(ctx, AgentWorkingScheduleService_SearchAgentsWorkingSchedule_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AgentWorkingScheduleServiceServer is the server API for AgentWorkingScheduleService service.
// All implementations must embed UnimplementedAgentWorkingScheduleServiceServer
// for forward compatibility
type AgentWorkingScheduleServiceServer interface {
	CreateAgentsWorkingScheduleShifts(context.Context, *CreateAgentsWorkingScheduleShiftsRequest) (*CreateAgentsWorkingScheduleShiftsResponse, error)
	SearchAgentsWorkingSchedule(context.Context, *SearchAgentsWorkingScheduleRequest) (*SearchAgentsWorkingScheduleResponse, error)
	mustEmbedUnimplementedAgentWorkingScheduleServiceServer()
}

// UnimplementedAgentWorkingScheduleServiceServer must be embedded to have forward compatible implementations.
type UnimplementedAgentWorkingScheduleServiceServer struct {
}

func (UnimplementedAgentWorkingScheduleServiceServer) CreateAgentsWorkingScheduleShifts(context.Context, *CreateAgentsWorkingScheduleShiftsRequest) (*CreateAgentsWorkingScheduleShiftsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateAgentsWorkingScheduleShifts not implemented")
}
func (UnimplementedAgentWorkingScheduleServiceServer) SearchAgentsWorkingSchedule(context.Context, *SearchAgentsWorkingScheduleRequest) (*SearchAgentsWorkingScheduleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SearchAgentsWorkingSchedule not implemented")
}
func (UnimplementedAgentWorkingScheduleServiceServer) mustEmbedUnimplementedAgentWorkingScheduleServiceServer() {
}

// UnsafeAgentWorkingScheduleServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AgentWorkingScheduleServiceServer will
// result in compilation errors.
type UnsafeAgentWorkingScheduleServiceServer interface {
	mustEmbedUnimplementedAgentWorkingScheduleServiceServer()
}

func RegisterAgentWorkingScheduleServiceServer(s grpc.ServiceRegistrar, srv AgentWorkingScheduleServiceServer) {
	s.RegisterService(&AgentWorkingScheduleService_ServiceDesc, srv)
}

func _AgentWorkingScheduleService_CreateAgentsWorkingScheduleShifts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateAgentsWorkingScheduleShiftsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentWorkingScheduleServiceServer).CreateAgentsWorkingScheduleShifts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: AgentWorkingScheduleService_CreateAgentsWorkingScheduleShifts_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentWorkingScheduleServiceServer).CreateAgentsWorkingScheduleShifts(ctx, req.(*CreateAgentsWorkingScheduleShiftsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AgentWorkingScheduleService_SearchAgentsWorkingSchedule_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SearchAgentsWorkingScheduleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentWorkingScheduleServiceServer).SearchAgentsWorkingSchedule(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: AgentWorkingScheduleService_SearchAgentsWorkingSchedule_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentWorkingScheduleServiceServer).SearchAgentsWorkingSchedule(ctx, req.(*SearchAgentsWorkingScheduleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// AgentWorkingScheduleService_ServiceDesc is the grpc.ServiceDesc for AgentWorkingScheduleService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var AgentWorkingScheduleService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "wfm.AgentWorkingScheduleService",
	HandlerType: (*AgentWorkingScheduleServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateAgentsWorkingScheduleShifts",
			Handler:    _AgentWorkingScheduleService_CreateAgentsWorkingScheduleShifts_Handler,
		},
		{
			MethodName: "SearchAgentsWorkingSchedule",
			Handler:    _AgentWorkingScheduleService_SearchAgentsWorkingSchedule_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "agent_working_schedule.proto",
}
