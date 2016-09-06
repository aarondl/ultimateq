// Code generated by protoc-gen-go.
// source: data.proto
// DO NOT EDIT!

/*
Package api is a generated protocol buffer package.

It is generated from these files:
	data.proto
	ultimateq.proto

It has these top-level messages:
	SimpleUser
	Handler
	Command
	ChannelModes
	ModeKinds
	StoredUser
	StoredChannel
	Access
	SelfResponse
	Empty
	Query
	NetworkQuery
	ChannelQuery
	ListResponse
	CountResponse
	Result
	RegisterRequest
	UnregisterRequest
	UserResponse
	UserModesResponse
	ChannelResponse
	StoredUsersResponse
	StoredChannelsResponse
	LogoutRequest
	IRCMessage
	NetworkInfo
	AuthUserRequest
	RawIRC
	ConnectionDetails
*/
package api

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type SimpleUser struct {
	Host string `protobuf:"bytes,1,opt,name=host" json:"host,omitempty"`
	Name string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
}

func (m *SimpleUser) Reset()                    { *m = SimpleUser{} }
func (m *SimpleUser) String() string            { return proto.CompactTextString(m) }
func (*SimpleUser) ProtoMessage()               {}
func (*SimpleUser) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type Handler struct {
	Network string `protobuf:"bytes,1,opt,name=network" json:"network,omitempty"`
	Channel string `protobuf:"bytes,2,opt,name=channel" json:"channel,omitempty"`
	Event   string `protobuf:"bytes,3,opt,name=event" json:"event,omitempty"`
}

func (m *Handler) Reset()                    { *m = Handler{} }
func (m *Handler) String() string            { return proto.CompactTextString(m) }
func (*Handler) ProtoMessage()               {}
func (*Handler) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

type Command struct {
	Network string       `protobuf:"bytes,1,opt,name=network" json:"network,omitempty"`
	Channel string       `protobuf:"bytes,2,opt,name=channel" json:"channel,omitempty"`
	Cmd     *Command_Cmd `protobuf:"bytes,3,opt,name=cmd" json:"cmd,omitempty"`
}

func (m *Command) Reset()                    { *m = Command{} }
func (m *Command) String() string            { return proto.CompactTextString(m) }
func (*Command) ProtoMessage()               {}
func (*Command) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *Command) GetCmd() *Command_Cmd {
	if m != nil {
		return m.Cmd
	}
	return nil
}

type Command_Cmd struct {
	Network     string   `protobuf:"bytes,1,opt,name=network" json:"network,omitempty"`
	Channel     string   `protobuf:"bytes,2,opt,name=channel" json:"channel,omitempty"`
	Description string   `protobuf:"bytes,3,opt,name=description" json:"description,omitempty"`
	Kind        int32    `protobuf:"varint,4,opt,name=kind" json:"kind,omitempty"`
	Scope       int32    `protobuf:"varint,5,opt,name=scope" json:"scope,omitempty"`
	Args        []string `protobuf:"bytes,6,rep,name=args" json:"args,omitempty"`
	RequireAuth bool     `protobuf:"varint,7,opt,name=require_auth,json=requireAuth" json:"require_auth,omitempty"`
	ReqLevel    int32    `protobuf:"varint,8,opt,name=req_level,json=reqLevel" json:"req_level,omitempty"`
	ReqFlags    string   `protobuf:"bytes,9,opt,name=req_flags,json=reqFlags" json:"req_flags,omitempty"`
}

func (m *Command_Cmd) Reset()                    { *m = Command_Cmd{} }
func (m *Command_Cmd) String() string            { return proto.CompactTextString(m) }
func (*Command_Cmd) ProtoMessage()               {}
func (*Command_Cmd) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2, 0} }

type ChannelModes struct {
	Modes        map[string]bool                      `protobuf:"bytes,1,rep,name=modes" json:"modes,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"varint,2,opt,name=value"`
	ArgModes     map[string]string                    `protobuf:"bytes,2,rep,name=arg_modes,json=argModes" json:"arg_modes,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	AddressModes map[string]*ChannelModes_AddressMode `protobuf:"bytes,3,rep,name=address_modes,json=addressModes" json:"address_modes,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	Addresses    int32                                `protobuf:"varint,4,opt,name=addresses" json:"addresses,omitempty"`
	Kinds        *ModeKinds                           `protobuf:"bytes,5,opt,name=kinds" json:"kinds,omitempty"`
}

func (m *ChannelModes) Reset()                    { *m = ChannelModes{} }
func (m *ChannelModes) String() string            { return proto.CompactTextString(m) }
func (*ChannelModes) ProtoMessage()               {}
func (*ChannelModes) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *ChannelModes) GetModes() map[string]bool {
	if m != nil {
		return m.Modes
	}
	return nil
}

func (m *ChannelModes) GetArgModes() map[string]string {
	if m != nil {
		return m.ArgModes
	}
	return nil
}

func (m *ChannelModes) GetAddressModes() map[string]*ChannelModes_AddressMode {
	if m != nil {
		return m.AddressModes
	}
	return nil
}

func (m *ChannelModes) GetKinds() *ModeKinds {
	if m != nil {
		return m.Kinds
	}
	return nil
}

type ChannelModes_AddressMode struct {
	ModeAddresses []string `protobuf:"bytes,1,rep,name=mode_addresses,json=modeAddresses" json:"mode_addresses,omitempty"`
}

func (m *ChannelModes_AddressMode) Reset()                    { *m = ChannelModes_AddressMode{} }
func (m *ChannelModes_AddressMode) String() string            { return proto.CompactTextString(m) }
func (*ChannelModes_AddressMode) ProtoMessage()               {}
func (*ChannelModes_AddressMode) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3, 0} }

type ModeKinds struct {
	UserPrefixes []*ModeKinds_UserPrefix `protobuf:"bytes,1,rep,name=user_prefixes,json=userPrefixes" json:"user_prefixes,omitempty"`
	ChannelModes map[string]int32        `protobuf:"bytes,2,rep,name=channel_modes,json=channelModes" json:"channel_modes,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"varint,2,opt,name=value"`
}

func (m *ModeKinds) Reset()                    { *m = ModeKinds{} }
func (m *ModeKinds) String() string            { return proto.CompactTextString(m) }
func (*ModeKinds) ProtoMessage()               {}
func (*ModeKinds) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *ModeKinds) GetUserPrefixes() []*ModeKinds_UserPrefix {
	if m != nil {
		return m.UserPrefixes
	}
	return nil
}

func (m *ModeKinds) GetChannelModes() map[string]int32 {
	if m != nil {
		return m.ChannelModes
	}
	return nil
}

type ModeKinds_UserPrefix struct {
	Symbol string `protobuf:"bytes,1,opt,name=symbol" json:"symbol,omitempty"`
	Char   string `protobuf:"bytes,2,opt,name=char" json:"char,omitempty"`
}

func (m *ModeKinds_UserPrefix) Reset()                    { *m = ModeKinds_UserPrefix{} }
func (m *ModeKinds_UserPrefix) String() string            { return proto.CompactTextString(m) }
func (*ModeKinds_UserPrefix) ProtoMessage()               {}
func (*ModeKinds_UserPrefix) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4, 0} }

type StoredUser struct {
	Username string             `protobuf:"bytes,1,opt,name=username" json:"username,omitempty"`
	Password string             `protobuf:"bytes,2,opt,name=password" json:"password,omitempty"`
	Masks    []string           `protobuf:"bytes,3,rep,name=masks" json:"masks,omitempty"`
	Access   map[string]*Access `protobuf:"bytes,4,rep,name=access" json:"access,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	Data     map[string]string  `protobuf:"bytes,5,rep,name=data" json:"data,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *StoredUser) Reset()                    { *m = StoredUser{} }
func (m *StoredUser) String() string            { return proto.CompactTextString(m) }
func (*StoredUser) ProtoMessage()               {}
func (*StoredUser) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *StoredUser) GetAccess() map[string]*Access {
	if m != nil {
		return m.Access
	}
	return nil
}

func (m *StoredUser) GetData() map[string]string {
	if m != nil {
		return m.Data
	}
	return nil
}

type StoredChannel struct {
	Network string            `protobuf:"bytes,1,opt,name=network" json:"network,omitempty"`
	Name    string            `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
	Data    map[string]string `protobuf:"bytes,3,rep,name=data" json:"data,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *StoredChannel) Reset()                    { *m = StoredChannel{} }
func (m *StoredChannel) String() string            { return proto.CompactTextString(m) }
func (*StoredChannel) ProtoMessage()               {}
func (*StoredChannel) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *StoredChannel) GetData() map[string]string {
	if m != nil {
		return m.Data
	}
	return nil
}

type Access struct {
	Level int32  `protobuf:"varint,1,opt,name=level" json:"level,omitempty"`
	Flags string `protobuf:"bytes,2,opt,name=flags" json:"flags,omitempty"`
}

func (m *Access) Reset()                    { *m = Access{} }
func (m *Access) String() string            { return proto.CompactTextString(m) }
func (*Access) ProtoMessage()               {}
func (*Access) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

func init() {
	proto.RegisterType((*SimpleUser)(nil), "api.SimpleUser")
	proto.RegisterType((*Handler)(nil), "api.Handler")
	proto.RegisterType((*Command)(nil), "api.Command")
	proto.RegisterType((*Command_Cmd)(nil), "api.Command.Cmd")
	proto.RegisterType((*ChannelModes)(nil), "api.ChannelModes")
	proto.RegisterType((*ChannelModes_AddressMode)(nil), "api.ChannelModes.AddressMode")
	proto.RegisterType((*ModeKinds)(nil), "api.ModeKinds")
	proto.RegisterType((*ModeKinds_UserPrefix)(nil), "api.ModeKinds.UserPrefix")
	proto.RegisterType((*StoredUser)(nil), "api.StoredUser")
	proto.RegisterType((*StoredChannel)(nil), "api.StoredChannel")
	proto.RegisterType((*Access)(nil), "api.Access")
}

func init() { proto.RegisterFile("data.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 746 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x9c, 0x55, 0x6f, 0x6b, 0x13, 0x4f,
	0x10, 0xe6, 0x72, 0xf9, 0x77, 0x73, 0x49, 0x69, 0x97, 0x1f, 0x3f, 0xce, 0xb4, 0x62, 0x1a, 0x15,
	0xf2, 0xc6, 0x20, 0x69, 0xc1, 0xa2, 0xa2, 0x84, 0xd8, 0x52, 0x50, 0x41, 0xae, 0xf8, 0xd6, 0xb0,
	0xbd, 0xdb, 0x26, 0x47, 0xee, 0x5f, 0x77, 0x2f, 0xad, 0xfd, 0x10, 0x7e, 0x02, 0xbf, 0x80, 0xdf,
	0xc2, 0xef, 0xe1, 0x07, 0xf1, 0xb5, 0xcc, 0xee, 0x26, 0xd9, 0x98, 0xd0, 0x62, 0xdf, 0x84, 0x99,
	0xd9, 0x79, 0x9e, 0x9d, 0x79, 0x76, 0x6e, 0x02, 0x10, 0xd2, 0x82, 0xf6, 0x72, 0x9e, 0x15, 0x19,
	0xb1, 0x69, 0x1e, 0x75, 0x0e, 0x01, 0xce, 0xa2, 0x24, 0x8f, 0xd9, 0x67, 0xc1, 0x38, 0x21, 0x50,
	0x9e, 0x64, 0xa2, 0xf0, 0xac, 0xb6, 0xd5, 0x75, 0x7c, 0x69, 0x63, 0x2c, 0xa5, 0x09, 0xf3, 0x4a,
	0x2a, 0x86, 0x76, 0xe7, 0x0c, 0x6a, 0xa7, 0x34, 0x0d, 0x63, 0xc6, 0x89, 0x07, 0xb5, 0x94, 0x15,
	0xd7, 0x19, 0x9f, 0x6a, 0xd4, 0xdc, 0xc5, 0x93, 0x60, 0x42, 0xd3, 0x94, 0xc5, 0x1a, 0x3b, 0x77,
	0xc9, 0x7f, 0x50, 0x61, 0x57, 0x2c, 0x2d, 0x3c, 0x5b, 0xc6, 0x95, 0xd3, 0xf9, 0x55, 0x82, 0xda,
	0x30, 0x4b, 0x12, 0x9a, 0x86, 0xf7, 0x62, 0xed, 0x80, 0x1d, 0x24, 0xa1, 0xe4, 0x74, 0xfb, 0xdb,
	0x3d, 0x9a, 0x47, 0x3d, 0x4d, 0xd7, 0x1b, 0x26, 0xa1, 0x8f, 0x87, 0xad, 0xdf, 0x16, 0xd8, 0xc3,
	0xe4, 0x7e, 0xfc, 0x6d, 0x70, 0x43, 0x26, 0x02, 0x1e, 0xe5, 0x45, 0x94, 0xa5, 0xba, 0x76, 0x33,
	0x84, 0x52, 0x4d, 0xa3, 0x34, 0xf4, 0xca, 0x6d, 0xab, 0x5b, 0xf1, 0xa5, 0x8d, 0xbd, 0x8a, 0x20,
	0xcb, 0x99, 0x57, 0x91, 0x41, 0xe5, 0x60, 0x26, 0xe5, 0x63, 0xe1, 0x55, 0xdb, 0x36, 0x8a, 0x8a,
	0x36, 0xd9, 0x87, 0x06, 0x67, 0x97, 0xb3, 0x88, 0xb3, 0x11, 0x9d, 0x15, 0x13, 0xaf, 0xd6, 0xb6,
	0xba, 0x75, 0xdf, 0xd5, 0xb1, 0xc1, 0xac, 0x98, 0x90, 0x5d, 0x70, 0x38, 0xbb, 0x1c, 0xc5, 0xec,
	0x8a, 0xc5, 0x5e, 0x5d, 0x12, 0xd6, 0x39, 0xbb, 0xfc, 0x80, 0xfe, 0xfc, 0xf0, 0x22, 0xa6, 0x63,
	0xe1, 0x39, 0xb2, 0x3a, 0x3c, 0x3c, 0x41, 0xbf, 0xf3, 0xbd, 0x0c, 0x8d, 0xa1, 0x6a, 0xe4, 0x63,
	0x16, 0x32, 0x41, 0xfa, 0x50, 0x49, 0xd0, 0xf0, 0xac, 0xb6, 0xdd, 0x75, 0xfb, 0x7b, 0x4a, 0x2f,
	0x23, 0xa3, 0x27, 0x7f, 0x8f, 0xd3, 0x82, 0xdf, 0xf8, 0x2a, 0x95, 0xbc, 0x06, 0x87, 0xf2, 0xf1,
	0x48, 0xe1, 0x4a, 0x12, 0xf7, 0x68, 0x1d, 0x37, 0xe0, 0x63, 0x03, 0x5a, 0xa7, 0xda, 0x25, 0xa7,
	0xd0, 0xa4, 0x61, 0xc8, 0x99, 0x10, 0x9a, 0xc1, 0x96, 0x0c, 0x8f, 0x37, 0x30, 0xa8, 0x34, 0x83,
	0xa5, 0x41, 0x8d, 0x10, 0xd9, 0x03, 0x47, 0xfb, 0x4c, 0x68, 0xb1, 0x97, 0x01, 0xf2, 0x04, 0x2a,
	0xa8, 0xbc, 0x90, 0x8a, 0xbb, 0xfd, 0x2d, 0xc9, 0x8f, 0xc0, 0xf7, 0x18, 0xf5, 0xd5, 0x61, 0xeb,
	0x10, 0x5c, 0xe3, 0x1a, 0xf2, 0x14, 0xb6, 0xb0, 0xa8, 0xd1, 0x92, 0xd7, 0x92, 0x4f, 0xd3, 0xc4,
	0xe8, 0x60, 0x1e, 0x6c, 0x1d, 0x01, 0x2c, 0xab, 0x22, 0xdb, 0x60, 0x4f, 0xd9, 0x8d, 0x9e, 0x20,
	0x34, 0xf1, 0xb5, 0xaf, 0x68, 0x3c, 0x53, 0x5f, 0x4b, 0xdd, 0x57, 0xce, 0xcb, 0xd2, 0x91, 0xd5,
	0x7a, 0x05, 0xcd, 0x15, 0x61, 0xee, 0x02, 0x3b, 0x26, 0xf8, 0x0b, 0xec, 0xac, 0x69, 0xb2, 0x81,
	0xe0, 0xc0, 0x24, 0x70, 0xfb, 0x0f, 0x6f, 0x55, 0xd6, 0xe0, 0xef, 0x7c, 0x2b, 0x81, 0xb3, 0x50,
	0x88, 0xbc, 0x81, 0xe6, 0x4c, 0x30, 0x3e, 0xca, 0x39, 0xbb, 0x88, 0xbe, 0x2e, 0x46, 0xe4, 0xc1,
	0xaa, 0x90, 0x3d, 0xdc, 0x18, 0x9f, 0x64, 0x8a, 0xdf, 0x98, 0x2d, 0x6c, 0x26, 0xc8, 0x31, 0x34,
	0xf5, 0x37, 0xb3, 0x32, 0x2a, 0xed, 0xbf, 0xf0, 0x66, 0x61, 0xfa, 0x95, 0x03, 0x23, 0x84, 0x5a,
	0x2f, 0xaf, 0x20, 0xff, 0x43, 0x55, 0xdc, 0x24, 0xe7, 0x59, 0xac, 0x1b, 0xd6, 0x1e, 0x7e, 0x49,
	0xc1, 0x84, 0xf2, 0xf9, 0x7a, 0x42, 0xbb, 0xf5, 0x16, 0x76, 0xd6, 0xc8, 0xef, 0xd2, 0xbb, 0x62,
	0xea, 0xf1, 0xb3, 0x04, 0x70, 0x56, 0x64, 0x9c, 0x85, 0x72, 0x2d, 0xb6, 0xa0, 0x8e, 0x0d, 0xca,
	0x35, 0xa8, 0xf0, 0x0b, 0x1f, 0xcf, 0x72, 0x2a, 0xc4, 0x75, 0xc6, 0x43, 0x5d, 0xc3, 0xc2, 0xc7,
	0x0b, 0x12, 0x2a, 0xa6, 0x6a, 0xd2, 0x1d, 0x5f, 0x39, 0xe4, 0x00, 0xaa, 0x34, 0x08, 0x98, 0xc0,
	0xd1, 0x45, 0x5d, 0x76, 0xa5, 0x2e, 0xcb, 0xeb, 0x7a, 0x03, 0x79, 0xaa, 0x24, 0xd1, 0xa9, 0xe4,
	0x19, 0x94, 0x71, 0x75, 0x7b, 0x15, 0xe3, 0x29, 0x0c, 0xc8, 0x3b, 0x5a, 0x50, 0x05, 0x90, 0x69,
	0xad, 0x13, 0x70, 0x0d, 0x96, 0x0d, 0xbd, 0xef, 0xaf, 0x8e, 0x8a, 0x2b, 0x09, 0x15, 0xc4, 0x1c,
	0xbc, 0x17, 0xe0, 0x2c, 0xa8, 0xff, 0x65, 0x62, 0x3b, 0x3f, 0x2c, 0x68, 0xaa, 0xfa, 0xf4, 0x4b,
	0xdc, 0xb2, 0x72, 0x37, 0xfc, 0xc3, 0x90, 0xe7, 0xba, 0x5f, 0xdb, 0xd8, 0x4e, 0x2b, 0x7c, 0x6b,
	0x2d, 0xdf, 0xbb, 0xd4, 0x43, 0xa8, 0xaa, 0xc6, 0x31, 0x47, 0xad, 0x56, 0x4b, 0x0d, 0x84, 0x74,
	0x30, 0xaa, 0x76, 0xaa, 0x46, 0x4a, 0xe7, 0xbc, 0x2a, 0xff, 0x44, 0x0f, 0xfe, 0x04, 0x00, 0x00,
	0xff, 0xff, 0x24, 0x8a, 0xd0, 0x5e, 0x52, 0x07, 0x00, 0x00,
}
