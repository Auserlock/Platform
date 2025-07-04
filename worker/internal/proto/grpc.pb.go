// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v6.31.1
// source: proto/grpc.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type CMDType int32

const (
	CMDType_QEMUVM      CMDType = 0
	CMDType_QEMUMonitor CMDType = 1
	CMDType_MCPServer   CMDType = 2
)

// Enum value maps for CMDType.
var (
	CMDType_name = map[int32]string{
		0: "QEMUVM",
		1: "QEMUMonitor",
		2: "MCPServer",
	}
	CMDType_value = map[string]int32{
		"QEMUVM":      0,
		"QEMUMonitor": 1,
		"MCPServer":   2,
	}
)

func (x CMDType) Enum() *CMDType {
	p := new(CMDType)
	*p = x
	return p
}

func (x CMDType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (CMDType) Descriptor() protoreflect.EnumDescriptor {
	return file_proto_grpc_proto_enumTypes[0].Descriptor()
}

func (CMDType) Type() protoreflect.EnumType {
	return &file_proto_grpc_proto_enumTypes[0]
}

func (x CMDType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use CMDType.Descriptor instead.
func (CMDType) EnumDescriptor() ([]byte, []int) {
	return file_proto_grpc_proto_rawDescGZIP(), []int{0}
}

type CommandType int32

const (
	CommandType_COMMAND_TYPE_UNSPECIFIED CommandType = 0
	CommandType_PONG                     CommandType = 1
	CommandType_OPEN_SSH                 CommandType = 2
	CommandType_OPEN_QEMU_MONITOR        CommandType = 3
	CommandType_EXECUTE_SHELL            CommandType = 4
	CommandType_RESTART_SERVICE          CommandType = 5
	CommandType_CUSTOM                   CommandType = 6
)

// Enum value maps for CommandType.
var (
	CommandType_name = map[int32]string{
		0: "COMMAND_TYPE_UNSPECIFIED",
		1: "PONG",
		2: "OPEN_SSH",
		3: "OPEN_QEMU_MONITOR",
		4: "EXECUTE_SHELL",
		5: "RESTART_SERVICE",
		6: "CUSTOM",
	}
	CommandType_value = map[string]int32{
		"COMMAND_TYPE_UNSPECIFIED": 0,
		"PONG":                     1,
		"OPEN_SSH":                 2,
		"OPEN_QEMU_MONITOR":        3,
		"EXECUTE_SHELL":            4,
		"RESTART_SERVICE":          5,
		"CUSTOM":                   6,
	}
)

func (x CommandType) Enum() *CommandType {
	p := new(CommandType)
	*p = x
	return p
}

func (x CommandType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (CommandType) Descriptor() protoreflect.EnumDescriptor {
	return file_proto_grpc_proto_enumTypes[1].Descriptor()
}

func (CommandType) Type() protoreflect.EnumType {
	return &file_proto_grpc_proto_enumTypes[1]
}

func (x CommandType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use CommandType.Descriptor instead.
func (CommandType) EnumDescriptor() ([]byte, []int) {
	return file_proto_grpc_proto_rawDescGZIP(), []int{1}
}

type CMDLine struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ClientId string  `protobuf:"bytes,1,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
	Type     CMDType `protobuf:"varint,2,opt,name=type,proto3,enum=grpc.CMDType" json:"type,omitempty"`
	Msg      string  `protobuf:"bytes,3,opt,name=msg,proto3" json:"msg,omitempty"`
}

func (x *CMDLine) Reset() {
	*x = CMDLine{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_grpc_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CMDLine) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CMDLine) ProtoMessage() {}

func (x *CMDLine) ProtoReflect() protoreflect.Message {
	mi := &file_proto_grpc_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CMDLine.ProtoReflect.Descriptor instead.
func (*CMDLine) Descriptor() ([]byte, []int) {
	return file_proto_grpc_proto_rawDescGZIP(), []int{0}
}

func (x *CMDLine) GetClientId() string {
	if x != nil {
		return x.ClientId
	}
	return ""
}

func (x *CMDLine) GetType() CMDType {
	if x != nil {
		return x.Type
	}
	return CMDType_QEMUVM
}

func (x *CMDLine) GetMsg() string {
	if x != nil {
		return x.Msg
	}
	return ""
}

type CMDCommand struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ClientId string  `protobuf:"bytes,1,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
	Type     CMDType `protobuf:"varint,2,opt,name=type,proto3,enum=grpc.CMDType" json:"type,omitempty"`
	Command  string  `protobuf:"bytes,3,opt,name=command,proto3" json:"command,omitempty"`
}

func (x *CMDCommand) Reset() {
	*x = CMDCommand{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_grpc_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CMDCommand) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CMDCommand) ProtoMessage() {}

func (x *CMDCommand) ProtoReflect() protoreflect.Message {
	mi := &file_proto_grpc_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CMDCommand.ProtoReflect.Descriptor instead.
func (*CMDCommand) Descriptor() ([]byte, []int) {
	return file_proto_grpc_proto_rawDescGZIP(), []int{1}
}

func (x *CMDCommand) GetClientId() string {
	if x != nil {
		return x.ClientId
	}
	return ""
}

func (x *CMDCommand) GetType() CMDType {
	if x != nil {
		return x.Type
	}
	return CMDType_QEMUVM
}

func (x *CMDCommand) GetCommand() string {
	if x != nil {
		return x.Command
	}
	return ""
}

type LogMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ClientId  string `protobuf:"bytes,1,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
	TaskId    string `protobuf:"bytes,2,opt,name=task_id,json=taskId,proto3" json:"task_id,omitempty"`
	Timestamp string `protobuf:"bytes,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Message   string `protobuf:"bytes,4,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *LogMessage) Reset() {
	*x = LogMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_grpc_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LogMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LogMessage) ProtoMessage() {}

func (x *LogMessage) ProtoReflect() protoreflect.Message {
	mi := &file_proto_grpc_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LogMessage.ProtoReflect.Descriptor instead.
func (*LogMessage) Descriptor() ([]byte, []int) {
	return file_proto_grpc_proto_rawDescGZIP(), []int{2}
}

func (x *LogMessage) GetClientId() string {
	if x != nil {
		return x.ClientId
	}
	return ""
}

func (x *LogMessage) GetTaskId() string {
	if x != nil {
		return x.TaskId
	}
	return ""
}

func (x *LogMessage) GetTimestamp() string {
	if x != nil {
		return x.Timestamp
	}
	return ""
}

func (x *LogMessage) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

type UploadLogsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Success bool   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *UploadLogsResponse) Reset() {
	*x = UploadLogsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_grpc_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UploadLogsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UploadLogsResponse) ProtoMessage() {}

func (x *UploadLogsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_grpc_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UploadLogsResponse.ProtoReflect.Descriptor instead.
func (*UploadLogsResponse) Descriptor() ([]byte, []int) {
	return file_proto_grpc_proto_rawDescGZIP(), []int{3}
}

func (x *UploadLogsResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *UploadLogsResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

type PingCommand struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ClientId string `protobuf:"bytes,1,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
	Time     string `protobuf:"bytes,2,opt,name=time,proto3" json:"time,omitempty"`
}

func (x *PingCommand) Reset() {
	*x = PingCommand{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_grpc_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PingCommand) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PingCommand) ProtoMessage() {}

func (x *PingCommand) ProtoReflect() protoreflect.Message {
	mi := &file_proto_grpc_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PingCommand.ProtoReflect.Descriptor instead.
func (*PingCommand) Descriptor() ([]byte, []int) {
	return file_proto_grpc_proto_rawDescGZIP(), []int{4}
}

func (x *PingCommand) GetClientId() string {
	if x != nil {
		return x.ClientId
	}
	return ""
}

func (x *PingCommand) GetTime() string {
	if x != nil {
		return x.Time
	}
	return ""
}

type Command struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CommandId          string            `protobuf:"bytes,1,opt,name=command_id,json=commandId,proto3" json:"command_id,omitempty"`
	Type               CommandType       `protobuf:"varint,2,opt,name=type,proto3,enum=grpc.CommandType" json:"type,omitempty"`
	TargetClient       string            `protobuf:"bytes,3,opt,name=target_client,json=targetClient,proto3" json:"target_client,omitempty"`
	Payload            string            `protobuf:"bytes,4,opt,name=payload,proto3" json:"payload,omitempty"`
	Params             map[string]string `protobuf:"bytes,5,rep,name=params,proto3" json:"params,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	NoCommandAvailable bool              `protobuf:"varint,6,opt,name=no_command_available,json=noCommandAvailable,proto3" json:"no_command_available,omitempty"`
}

func (x *Command) Reset() {
	*x = Command{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_grpc_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Command) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Command) ProtoMessage() {}

func (x *Command) ProtoReflect() protoreflect.Message {
	mi := &file_proto_grpc_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Command.ProtoReflect.Descriptor instead.
func (*Command) Descriptor() ([]byte, []int) {
	return file_proto_grpc_proto_rawDescGZIP(), []int{5}
}

func (x *Command) GetCommandId() string {
	if x != nil {
		return x.CommandId
	}
	return ""
}

func (x *Command) GetType() CommandType {
	if x != nil {
		return x.Type
	}
	return CommandType_COMMAND_TYPE_UNSPECIFIED
}

func (x *Command) GetTargetClient() string {
	if x != nil {
		return x.TargetClient
	}
	return ""
}

func (x *Command) GetPayload() string {
	if x != nil {
		return x.Payload
	}
	return ""
}

func (x *Command) GetParams() map[string]string {
	if x != nil {
		return x.Params
	}
	return nil
}

func (x *Command) GetNoCommandAvailable() bool {
	if x != nil {
		return x.NoCommandAvailable
	}
	return false
}

var File_proto_grpc_proto protoreflect.FileDescriptor

var file_proto_grpc_proto_rawDesc = []byte{
	0x0a, 0x10, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x04, 0x67, 0x72, 0x70, 0x63, 0x22, 0x5b, 0x0a, 0x07, 0x43, 0x4d, 0x44, 0x4c,
	0x69, 0x6e, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x49, 0x64,
	0x12, 0x21, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0d,
	0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x43, 0x4d, 0x44, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74,
	0x79, 0x70, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x6d, 0x73, 0x67, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x03, 0x6d, 0x73, 0x67, 0x22, 0x66, 0x0a, 0x0a, 0x43, 0x4d, 0x44, 0x43, 0x6f, 0x6d, 0x6d,
	0x61, 0x6e, 0x64, 0x12, 0x1b, 0x0a, 0x09, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x49, 0x64,
	0x12, 0x21, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0d,
	0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x43, 0x4d, 0x44, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74,
	0x79, 0x70, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x22, 0x7a, 0x0a,
	0x0a, 0x4c, 0x6f, 0x67, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x63,
	0x6c, 0x69, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08,
	0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x17, 0x0a, 0x07, 0x74, 0x61, 0x73, 0x6b,
	0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x74, 0x61, 0x73, 0x6b, 0x49,
	0x64, 0x12, 0x1c, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12,
	0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x22, 0x48, 0x0a, 0x12, 0x55, 0x70, 0x6c,
	0x6f, 0x61, 0x64, 0x4c, 0x6f, 0x67, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x18, 0x0a, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08,
	0x52, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x22, 0x3e, 0x0a, 0x0b, 0x50, 0x69, 0x6e, 0x67, 0x43, 0x6f, 0x6d, 0x6d, 0x61,
	0x6e, 0x64, 0x12, 0x1b, 0x0a, 0x09, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x12,
	0x12, 0x0a, 0x04, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74,
	0x69, 0x6d, 0x65, 0x22, 0xae, 0x02, 0x0a, 0x07, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x12,
	0x1d, 0x0a, 0x0a, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x09, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x49, 0x64, 0x12, 0x25,
	0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x11, 0x2e, 0x67,
	0x72, 0x70, 0x63, 0x2e, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x54, 0x79, 0x70, 0x65, 0x52,
	0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x23, 0x0a, 0x0d, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x5f,
	0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x74, 0x61,
	0x72, 0x67, 0x65, 0x74, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61,
	0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x70, 0x61, 0x79,
	0x6c, 0x6f, 0x61, 0x64, 0x12, 0x31, 0x0a, 0x06, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x18, 0x05,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x43, 0x6f, 0x6d, 0x6d,
	0x61, 0x6e, 0x64, 0x2e, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52,
	0x06, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x12, 0x30, 0x0a, 0x14, 0x6e, 0x6f, 0x5f, 0x63, 0x6f,
	0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x5f, 0x61, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x6c, 0x65, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x08, 0x52, 0x12, 0x6e, 0x6f, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64,
	0x41, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x6c, 0x65, 0x1a, 0x39, 0x0a, 0x0b, 0x50, 0x61, 0x72,
	0x61, 0x6d, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x3a, 0x02, 0x38, 0x01, 0x2a, 0x35, 0x0a, 0x07, 0x43, 0x4d, 0x44, 0x54, 0x79, 0x70, 0x65, 0x12,
	0x0a, 0x0a, 0x06, 0x51, 0x45, 0x4d, 0x55, 0x56, 0x4d, 0x10, 0x00, 0x12, 0x0f, 0x0a, 0x0b, 0x51,
	0x45, 0x4d, 0x55, 0x4d, 0x6f, 0x6e, 0x69, 0x74, 0x6f, 0x72, 0x10, 0x01, 0x12, 0x0d, 0x0a, 0x09,
	0x4d, 0x43, 0x50, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x10, 0x02, 0x2a, 0x8e, 0x01, 0x0a, 0x0b,
	0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x54, 0x79, 0x70, 0x65, 0x12, 0x1c, 0x0a, 0x18, 0x43,
	0x4f, 0x4d, 0x4d, 0x41, 0x4e, 0x44, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x55, 0x4e, 0x53, 0x50,
	0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x08, 0x0a, 0x04, 0x50, 0x4f, 0x4e,
	0x47, 0x10, 0x01, 0x12, 0x0c, 0x0a, 0x08, 0x4f, 0x50, 0x45, 0x4e, 0x5f, 0x53, 0x53, 0x48, 0x10,
	0x02, 0x12, 0x15, 0x0a, 0x11, 0x4f, 0x50, 0x45, 0x4e, 0x5f, 0x51, 0x45, 0x4d, 0x55, 0x5f, 0x4d,
	0x4f, 0x4e, 0x49, 0x54, 0x4f, 0x52, 0x10, 0x03, 0x12, 0x11, 0x0a, 0x0d, 0x45, 0x58, 0x45, 0x43,
	0x55, 0x54, 0x45, 0x5f, 0x53, 0x48, 0x45, 0x4c, 0x4c, 0x10, 0x04, 0x12, 0x13, 0x0a, 0x0f, 0x52,
	0x45, 0x53, 0x54, 0x41, 0x52, 0x54, 0x5f, 0x53, 0x45, 0x52, 0x56, 0x49, 0x43, 0x45, 0x10, 0x05,
	0x12, 0x0a, 0x0a, 0x06, 0x43, 0x55, 0x53, 0x54, 0x4f, 0x4d, 0x10, 0x06, 0x32, 0x4e, 0x0a, 0x10,
	0x4c, 0x6f, 0x67, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x12, 0x3a, 0x0a, 0x0a, 0x55, 0x70, 0x6c, 0x6f, 0x61, 0x64, 0x4c, 0x6f, 0x67, 0x73, 0x12, 0x10,
	0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x4c, 0x6f, 0x67, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x1a, 0x18, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x55, 0x70, 0x6c, 0x6f, 0x61, 0x64, 0x4c, 0x6f,
	0x67, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x28, 0x01, 0x32, 0x40, 0x0a, 0x0e,
	0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x2e,
	0x0a, 0x0a, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x12, 0x11, 0x2e, 0x67,
	0x72, 0x70, 0x63, 0x2e, 0x50, 0x69, 0x6e, 0x67, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x1a,
	0x0d, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x32, 0x41,
	0x0a, 0x10, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x70, 0x6f, 0x72, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x12, 0x2d, 0x0a, 0x06, 0x55, 0x70, 0x6c, 0x6f, 0x61, 0x64, 0x12, 0x0d, 0x2e, 0x67,
	0x72, 0x70, 0x63, 0x2e, 0x43, 0x4d, 0x44, 0x4c, 0x69, 0x6e, 0x65, 0x1a, 0x10, 0x2e, 0x67, 0x72,
	0x70, 0x63, 0x2e, 0x43, 0x4d, 0x44, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x28, 0x01, 0x30,
	0x01, 0x42, 0x10, 0x5a, 0x0e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_grpc_proto_rawDescOnce sync.Once
	file_proto_grpc_proto_rawDescData = file_proto_grpc_proto_rawDesc
)

func file_proto_grpc_proto_rawDescGZIP() []byte {
	file_proto_grpc_proto_rawDescOnce.Do(func() {
		file_proto_grpc_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_grpc_proto_rawDescData)
	})
	return file_proto_grpc_proto_rawDescData
}

var file_proto_grpc_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_proto_grpc_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_proto_grpc_proto_goTypes = []interface{}{
	(CMDType)(0),               // 0: grpc.CMDType
	(CommandType)(0),           // 1: grpc.CommandType
	(*CMDLine)(nil),            // 2: grpc.CMDLine
	(*CMDCommand)(nil),         // 3: grpc.CMDCommand
	(*LogMessage)(nil),         // 4: grpc.LogMessage
	(*UploadLogsResponse)(nil), // 5: grpc.UploadLogsResponse
	(*PingCommand)(nil),        // 6: grpc.PingCommand
	(*Command)(nil),            // 7: grpc.Command
	nil,                        // 8: grpc.Command.ParamsEntry
}
var file_proto_grpc_proto_depIdxs = []int32{
	0, // 0: grpc.CMDLine.type:type_name -> grpc.CMDType
	0, // 1: grpc.CMDCommand.type:type_name -> grpc.CMDType
	1, // 2: grpc.Command.type:type_name -> grpc.CommandType
	8, // 3: grpc.Command.params:type_name -> grpc.Command.ParamsEntry
	4, // 4: grpc.LogStreamService.UploadLogs:input_type -> grpc.LogMessage
	6, // 5: grpc.CommandService.GetCommand:input_type -> grpc.PingCommand
	2, // 6: grpc.TransportService.Upload:input_type -> grpc.CMDLine
	5, // 7: grpc.LogStreamService.UploadLogs:output_type -> grpc.UploadLogsResponse
	7, // 8: grpc.CommandService.GetCommand:output_type -> grpc.Command
	3, // 9: grpc.TransportService.Upload:output_type -> grpc.CMDCommand
	7, // [7:10] is the sub-list for method output_type
	4, // [4:7] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_proto_grpc_proto_init() }
func file_proto_grpc_proto_init() {
	if File_proto_grpc_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proto_grpc_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CMDLine); i {
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
		file_proto_grpc_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CMDCommand); i {
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
		file_proto_grpc_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LogMessage); i {
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
		file_proto_grpc_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UploadLogsResponse); i {
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
		file_proto_grpc_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PingCommand); i {
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
		file_proto_grpc_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Command); i {
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
			RawDescriptor: file_proto_grpc_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   3,
		},
		GoTypes:           file_proto_grpc_proto_goTypes,
		DependencyIndexes: file_proto_grpc_proto_depIdxs,
		EnumInfos:         file_proto_grpc_proto_enumTypes,
		MessageInfos:      file_proto_grpc_proto_msgTypes,
	}.Build()
	File_proto_grpc_proto = out.File
	file_proto_grpc_proto_rawDesc = nil
	file_proto_grpc_proto_goTypes = nil
	file_proto_grpc_proto_depIdxs = nil
}
