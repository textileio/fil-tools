// Code generated by protoc-gen-go. DO NOT EDIT.
// source: api.proto

package rpc

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type Status int32

const (
	Status_Ok       Status = 0
	Status_Degraded Status = 1
	Status_Error    Status = 2
)

var Status_name = map[int32]string{
	0: "Ok",
	1: "Degraded",
	2: "Error",
}

var Status_value = map[string]int32{
	"Ok":       0,
	"Degraded": 1,
	"Error":    2,
}

func (x Status) String() string {
	return proto.EnumName(Status_name, int32(x))
}

func (Status) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{0}
}

type CheckRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CheckRequest) Reset()         { *m = CheckRequest{} }
func (m *CheckRequest) String() string { return proto.CompactTextString(m) }
func (*CheckRequest) ProtoMessage()    {}
func (*CheckRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{0}
}

func (m *CheckRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CheckRequest.Unmarshal(m, b)
}
func (m *CheckRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CheckRequest.Marshal(b, m, deterministic)
}
func (m *CheckRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CheckRequest.Merge(m, src)
}
func (m *CheckRequest) XXX_Size() int {
	return xxx_messageInfo_CheckRequest.Size(m)
}
func (m *CheckRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_CheckRequest.DiscardUnknown(m)
}

var xxx_messageInfo_CheckRequest proto.InternalMessageInfo

type CheckReply struct {
	Status               Status   `protobuf:"varint,1,opt,name=status,proto3,enum=health.rpc.Status" json:"status,omitempty"`
	Messages             []string `protobuf:"bytes,2,rep,name=messages,proto3" json:"messages,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CheckReply) Reset()         { *m = CheckReply{} }
func (m *CheckReply) String() string { return proto.CompactTextString(m) }
func (*CheckReply) ProtoMessage()    {}
func (*CheckReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{1}
}

func (m *CheckReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CheckReply.Unmarshal(m, b)
}
func (m *CheckReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CheckReply.Marshal(b, m, deterministic)
}
func (m *CheckReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CheckReply.Merge(m, src)
}
func (m *CheckReply) XXX_Size() int {
	return xxx_messageInfo_CheckReply.Size(m)
}
func (m *CheckReply) XXX_DiscardUnknown() {
	xxx_messageInfo_CheckReply.DiscardUnknown(m)
}

var xxx_messageInfo_CheckReply proto.InternalMessageInfo

func (m *CheckReply) GetStatus() Status {
	if m != nil {
		return m.Status
	}
	return Status_Ok
}

func (m *CheckReply) GetMessages() []string {
	if m != nil {
		return m.Messages
	}
	return nil
}

func init() {
	proto.RegisterEnum("health.rpc.Status", Status_name, Status_value)
	proto.RegisterType((*CheckRequest)(nil), "health.rpc.CheckRequest")
	proto.RegisterType((*CheckReply)(nil), "health.rpc.CheckReply")
}

func init() { proto.RegisterFile("api.proto", fileDescriptor_00212fb1f9d3bf1c) }

var fileDescriptor_00212fb1f9d3bf1c = []byte{
	// 239 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x90, 0x31, 0x4b, 0xc3, 0x50,
	0x14, 0x85, 0x9b, 0x84, 0x86, 0xe6, 0x52, 0x6a, 0xb8, 0x83, 0x84, 0x2e, 0x96, 0x4c, 0xb5, 0xc3,
	0x1b, 0xea, 0xe8, 0x20, 0x46, 0x0b, 0x3a, 0x19, 0x62, 0x26, 0xb7, 0x67, 0x7a, 0x49, 0x42, 0x23,
	0xef, 0x79, 0xdf, 0x2d, 0xda, 0xbf, 0xe3, 0x2f, 0x15, 0x62, 0xd5, 0x0c, 0x1d, 0xcf, 0x39, 0xdf,
	0xe5, 0x1e, 0x0e, 0x44, 0xda, 0xb6, 0xca, 0xb2, 0x11, 0x83, 0xd0, 0x90, 0xee, 0xa4, 0x51, 0x6c,
	0xab, 0x74, 0x06, 0xd3, 0xbb, 0x86, 0xaa, 0x5d, 0x41, 0xef, 0x7b, 0x72, 0x92, 0x96, 0x00, 0x47,
	0x6d, 0xbb, 0x03, 0xae, 0x20, 0x74, 0xa2, 0x65, 0xef, 0x12, 0x6f, 0xe1, 0x2d, 0x67, 0x6b, 0x54,
	0xff, 0xa7, 0xea, 0xb9, 0x4f, 0x8a, 0x23, 0x81, 0x73, 0x98, 0xbc, 0x91, 0x73, 0xba, 0x26, 0x97,
	0xf8, 0x8b, 0x60, 0x19, 0x15, 0x7f, 0x7a, 0x75, 0x09, 0xe1, 0x0f, 0x8d, 0x21, 0xf8, 0x4f, 0xbb,
	0x78, 0x84, 0x53, 0x98, 0xdc, 0x53, 0xcd, 0x7a, 0x4b, 0xdb, 0xd8, 0xc3, 0x08, 0xc6, 0x1b, 0x66,
	0xc3, 0xb1, 0xbf, 0xce, 0x20, 0xb8, 0xcd, 0x1f, 0xf1, 0x1a, 0xc6, 0x7d, 0x0f, 0x4c, 0x86, 0x2f,
	0x87, 0x55, 0xe7, 0xe7, 0x27, 0x12, 0xdb, 0x1d, 0xd2, 0x51, 0x76, 0x03, 0x17, 0xad, 0x51, 0x42,
	0x9f, 0xd2, 0x76, 0xa4, 0xac, 0xf9, 0x20, 0xae, 0xb5, 0xd0, 0x80, 0xcf, 0xce, 0xf2, 0x5f, 0xf7,
	0xa1, 0x37, 0x73, 0xef, 0x25, 0x60, 0x5b, 0x7d, 0xf9, 0x41, 0x59, 0x6e, 0x5e, 0xc3, 0x7e, 0xa8,
	0xab, 0xef, 0x00, 0x00, 0x00, 0xff, 0xff, 0x68, 0x72, 0x28, 0xda, 0x35, 0x01, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// APIClient is the client API for API service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type APIClient interface {
	Check(ctx context.Context, in *CheckRequest, opts ...grpc.CallOption) (*CheckReply, error)
}

type aPIClient struct {
	cc *grpc.ClientConn
}

func NewAPIClient(cc *grpc.ClientConn) APIClient {
	return &aPIClient{cc}
}

func (c *aPIClient) Check(ctx context.Context, in *CheckRequest, opts ...grpc.CallOption) (*CheckReply, error) {
	out := new(CheckReply)
	err := c.cc.Invoke(ctx, "/health.rpc.API/Check", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// APIServer is the server API for API service.
type APIServer interface {
	Check(context.Context, *CheckRequest) (*CheckReply, error)
}

// UnimplementedAPIServer can be embedded to have forward compatible implementations.
type UnimplementedAPIServer struct {
}

func (*UnimplementedAPIServer) Check(ctx context.Context, req *CheckRequest) (*CheckReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Check not implemented")
}

func RegisterAPIServer(s *grpc.Server, srv APIServer) {
	s.RegisterService(&_API_serviceDesc, srv)
}

func _API_Check_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CheckRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(APIServer).Check(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/health.rpc.API/Check",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(APIServer).Check(ctx, req.(*CheckRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _API_serviceDesc = grpc.ServiceDesc{
	ServiceName: "health.rpc.API",
	HandlerType: (*APIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Check",
			Handler:    _API_Check_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api.proto",
}
