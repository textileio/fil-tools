// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package adminPb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// AdminServiceClient is the client API for AdminService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AdminServiceClient interface {
	// Wallet
	NewAddress(ctx context.Context, in *NewAddressRequest, opts ...grpc.CallOption) (*NewAddressResponse, error)
	Addresses(ctx context.Context, in *AddressesRequest, opts ...grpc.CallOption) (*AddressesResponse, error)
	SendFil(ctx context.Context, in *SendFilRequest, opts ...grpc.CallOption) (*SendFilResponse, error)
	// Users
	CreateUser(ctx context.Context, in *CreateUserRequest, opts ...grpc.CallOption) (*CreateUserResponse, error)
	Users(ctx context.Context, in *UsersRequest, opts ...grpc.CallOption) (*UsersResponse, error)
	// Jobs
	ListStorageJobs(ctx context.Context, in *ListStorageJobsRequest, opts ...grpc.CallOption) (*ListStorageJobsResponse, error)
	StorageJobsSummary(ctx context.Context, in *StorageJobsSummaryRequest, opts ...grpc.CallOption) (*StorageJobsSummaryResponse, error)
	GCStaged(ctx context.Context, in *GCStagedRequest, opts ...grpc.CallOption) (*GCStagedResponse, error)
	PinnedCids(ctx context.Context, in *PinnedCidsRequest, opts ...grpc.CallOption) (*PinnedCidsResponse, error)
}

type adminServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewAdminServiceClient(cc grpc.ClientConnInterface) AdminServiceClient {
	return &adminServiceClient{cc}
}

func (c *adminServiceClient) NewAddress(ctx context.Context, in *NewAddressRequest, opts ...grpc.CallOption) (*NewAddressResponse, error) {
	out := new(NewAddressResponse)
	err := c.cc.Invoke(ctx, "/powergate.admin.v1.AdminService/NewAddress", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adminServiceClient) Addresses(ctx context.Context, in *AddressesRequest, opts ...grpc.CallOption) (*AddressesResponse, error) {
	out := new(AddressesResponse)
	err := c.cc.Invoke(ctx, "/powergate.admin.v1.AdminService/Addresses", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adminServiceClient) SendFil(ctx context.Context, in *SendFilRequest, opts ...grpc.CallOption) (*SendFilResponse, error) {
	out := new(SendFilResponse)
	err := c.cc.Invoke(ctx, "/powergate.admin.v1.AdminService/SendFil", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adminServiceClient) CreateUser(ctx context.Context, in *CreateUserRequest, opts ...grpc.CallOption) (*CreateUserResponse, error) {
	out := new(CreateUserResponse)
	err := c.cc.Invoke(ctx, "/powergate.admin.v1.AdminService/CreateUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adminServiceClient) Users(ctx context.Context, in *UsersRequest, opts ...grpc.CallOption) (*UsersResponse, error) {
	out := new(UsersResponse)
	err := c.cc.Invoke(ctx, "/powergate.admin.v1.AdminService/Users", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adminServiceClient) ListStorageJobs(ctx context.Context, in *ListStorageJobsRequest, opts ...grpc.CallOption) (*ListStorageJobsResponse, error) {
	out := new(ListStorageJobsResponse)
	err := c.cc.Invoke(ctx, "/powergate.admin.v1.AdminService/ListStorageJobs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adminServiceClient) StorageJobsSummary(ctx context.Context, in *StorageJobsSummaryRequest, opts ...grpc.CallOption) (*StorageJobsSummaryResponse, error) {
	out := new(StorageJobsSummaryResponse)
	err := c.cc.Invoke(ctx, "/powergate.admin.v1.AdminService/StorageJobsSummary", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adminServiceClient) GCStaged(ctx context.Context, in *GCStagedRequest, opts ...grpc.CallOption) (*GCStagedResponse, error) {
	out := new(GCStagedResponse)
	err := c.cc.Invoke(ctx, "/powergate.admin.v1.AdminService/GCStaged", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adminServiceClient) PinnedCids(ctx context.Context, in *PinnedCidsRequest, opts ...grpc.CallOption) (*PinnedCidsResponse, error) {
	out := new(PinnedCidsResponse)
	err := c.cc.Invoke(ctx, "/powergate.admin.v1.AdminService/PinnedCids", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AdminServiceServer is the server API for AdminService service.
// All implementations must embed UnimplementedAdminServiceServer
// for forward compatibility
type AdminServiceServer interface {
	// Wallet
	NewAddress(context.Context, *NewAddressRequest) (*NewAddressResponse, error)
	Addresses(context.Context, *AddressesRequest) (*AddressesResponse, error)
	SendFil(context.Context, *SendFilRequest) (*SendFilResponse, error)
	// Users
	CreateUser(context.Context, *CreateUserRequest) (*CreateUserResponse, error)
	Users(context.Context, *UsersRequest) (*UsersResponse, error)
	// Jobs
	ListStorageJobs(context.Context, *ListStorageJobsRequest) (*ListStorageJobsResponse, error)
	StorageJobsSummary(context.Context, *StorageJobsSummaryRequest) (*StorageJobsSummaryResponse, error)
	GCStaged(context.Context, *GCStagedRequest) (*GCStagedResponse, error)
	PinnedCids(context.Context, *PinnedCidsRequest) (*PinnedCidsResponse, error)
	mustEmbedUnimplementedAdminServiceServer()
}

// UnimplementedAdminServiceServer must be embedded to have forward compatible implementations.
type UnimplementedAdminServiceServer struct {
}

func (UnimplementedAdminServiceServer) NewAddress(context.Context, *NewAddressRequest) (*NewAddressResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NewAddress not implemented")
}
func (UnimplementedAdminServiceServer) Addresses(context.Context, *AddressesRequest) (*AddressesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Addresses not implemented")
}
func (UnimplementedAdminServiceServer) SendFil(context.Context, *SendFilRequest) (*SendFilResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SendFil not implemented")
}
func (UnimplementedAdminServiceServer) CreateUser(context.Context, *CreateUserRequest) (*CreateUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateUser not implemented")
}
func (UnimplementedAdminServiceServer) Users(context.Context, *UsersRequest) (*UsersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Users not implemented")
}
func (UnimplementedAdminServiceServer) ListStorageJobs(context.Context, *ListStorageJobsRequest) (*ListStorageJobsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListStorageJobs not implemented")
}
func (UnimplementedAdminServiceServer) StorageJobsSummary(context.Context, *StorageJobsSummaryRequest) (*StorageJobsSummaryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StorageJobsSummary not implemented")
}
func (UnimplementedAdminServiceServer) GCStaged(context.Context, *GCStagedRequest) (*GCStagedResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GCStaged not implemented")
}
func (UnimplementedAdminServiceServer) PinnedCids(context.Context, *PinnedCidsRequest) (*PinnedCidsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PinnedCids not implemented")
}
func (UnimplementedAdminServiceServer) mustEmbedUnimplementedAdminServiceServer() {}

// UnsafeAdminServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AdminServiceServer will
// result in compilation errors.
type UnsafeAdminServiceServer interface {
	mustEmbedUnimplementedAdminServiceServer()
}

func RegisterAdminServiceServer(s grpc.ServiceRegistrar, srv AdminServiceServer) {
	s.RegisterService(&_AdminService_serviceDesc, srv)
}

func _AdminService_NewAddress_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NewAddressRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdminServiceServer).NewAddress(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/powergate.admin.v1.AdminService/NewAddress",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdminServiceServer).NewAddress(ctx, req.(*NewAddressRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdminService_Addresses_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddressesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdminServiceServer).Addresses(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/powergate.admin.v1.AdminService/Addresses",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdminServiceServer).Addresses(ctx, req.(*AddressesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdminService_SendFil_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SendFilRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdminServiceServer).SendFil(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/powergate.admin.v1.AdminService/SendFil",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdminServiceServer).SendFil(ctx, req.(*SendFilRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdminService_CreateUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdminServiceServer).CreateUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/powergate.admin.v1.AdminService/CreateUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdminServiceServer).CreateUser(ctx, req.(*CreateUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdminService_Users_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UsersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdminServiceServer).Users(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/powergate.admin.v1.AdminService/Users",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdminServiceServer).Users(ctx, req.(*UsersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdminService_ListStorageJobs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListStorageJobsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdminServiceServer).ListStorageJobs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/powergate.admin.v1.AdminService/ListStorageJobs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdminServiceServer).ListStorageJobs(ctx, req.(*ListStorageJobsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdminService_StorageJobsSummary_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StorageJobsSummaryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdminServiceServer).StorageJobsSummary(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/powergate.admin.v1.AdminService/StorageJobsSummary",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdminServiceServer).StorageJobsSummary(ctx, req.(*StorageJobsSummaryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdminService_GCStaged_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GCStagedRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdminServiceServer).GCStaged(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/powergate.admin.v1.AdminService/GCStaged",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdminServiceServer).GCStaged(ctx, req.(*GCStagedRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdminService_PinnedCids_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PinnedCidsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdminServiceServer).PinnedCids(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/powergate.admin.v1.AdminService/PinnedCids",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdminServiceServer).PinnedCids(ctx, req.(*PinnedCidsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _AdminService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "powergate.admin.v1.AdminService",
	HandlerType: (*AdminServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "NewAddress",
			Handler:    _AdminService_NewAddress_Handler,
		},
		{
			MethodName: "Addresses",
			Handler:    _AdminService_Addresses_Handler,
		},
		{
			MethodName: "SendFil",
			Handler:    _AdminService_SendFil_Handler,
		},
		{
			MethodName: "CreateUser",
			Handler:    _AdminService_CreateUser_Handler,
		},
		{
			MethodName: "Users",
			Handler:    _AdminService_Users_Handler,
		},
		{
			MethodName: "ListStorageJobs",
			Handler:    _AdminService_ListStorageJobs_Handler,
		},
		{
			MethodName: "StorageJobsSummary",
			Handler:    _AdminService_StorageJobsSummary_Handler,
		},
		{
			MethodName: "GCStaged",
			Handler:    _AdminService_GCStaged_Handler,
		},
		{
			MethodName: "PinnedCids",
			Handler:    _AdminService_PinnedCids_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "powergate/admin/v1/admin.proto",
}
