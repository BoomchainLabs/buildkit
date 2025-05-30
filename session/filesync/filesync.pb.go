// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v3.11.4
// source: github.com/moby/buildkit/session/filesync/filesync.proto

package filesync

import (
	types "github.com/tonistiigi/fsutil/types"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// BytesMessage contains a chunk of byte data
type BytesMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Data          []byte                 `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *BytesMessage) Reset() {
	*x = BytesMessage{}
	mi := &file_github_com_moby_buildkit_session_filesync_filesync_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BytesMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BytesMessage) ProtoMessage() {}

func (x *BytesMessage) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_moby_buildkit_session_filesync_filesync_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BytesMessage.ProtoReflect.Descriptor instead.
func (*BytesMessage) Descriptor() ([]byte, []int) {
	return file_github_com_moby_buildkit_session_filesync_filesync_proto_rawDescGZIP(), []int{0}
}

func (x *BytesMessage) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

var File_github_com_moby_buildkit_session_filesync_filesync_proto protoreflect.FileDescriptor

const file_github_com_moby_buildkit_session_filesync_filesync_proto_rawDesc = "" +
	"\n" +
	"8github.com/moby/buildkit/session/filesync/filesync.proto\x12\x10moby.filesync.v1\x1a-github.com/tonistiigi/fsutil/types/wire.proto\"\"\n" +
	"\fBytesMessage\x12\x12\n" +
	"\x04data\x18\x01 \x01(\fR\x04data2\x83\x01\n" +
	"\bFileSync\x12:\n" +
	"\bDiffCopy\x12\x14.fsutil.types.Packet\x1a\x14.fsutil.types.Packet(\x010\x01\x12;\n" +
	"\tTarStream\x12\x14.fsutil.types.Packet\x1a\x14.fsutil.types.Packet(\x010\x012Z\n" +
	"\bFileSend\x12N\n" +
	"\bDiffCopy\x12\x1e.moby.filesync.v1.BytesMessage\x1a\x1e.moby.filesync.v1.BytesMessage(\x010\x01B+Z)github.com/moby/buildkit/session/filesyncb\x06proto3"

var (
	file_github_com_moby_buildkit_session_filesync_filesync_proto_rawDescOnce sync.Once
	file_github_com_moby_buildkit_session_filesync_filesync_proto_rawDescData []byte
)

func file_github_com_moby_buildkit_session_filesync_filesync_proto_rawDescGZIP() []byte {
	file_github_com_moby_buildkit_session_filesync_filesync_proto_rawDescOnce.Do(func() {
		file_github_com_moby_buildkit_session_filesync_filesync_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_github_com_moby_buildkit_session_filesync_filesync_proto_rawDesc), len(file_github_com_moby_buildkit_session_filesync_filesync_proto_rawDesc)))
	})
	return file_github_com_moby_buildkit_session_filesync_filesync_proto_rawDescData
}

var file_github_com_moby_buildkit_session_filesync_filesync_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_moby_buildkit_session_filesync_filesync_proto_goTypes = []any{
	(*BytesMessage)(nil), // 0: moby.filesync.v1.BytesMessage
	(*types.Packet)(nil), // 1: fsutil.types.Packet
}
var file_github_com_moby_buildkit_session_filesync_filesync_proto_depIdxs = []int32{
	1, // 0: moby.filesync.v1.FileSync.DiffCopy:input_type -> fsutil.types.Packet
	1, // 1: moby.filesync.v1.FileSync.TarStream:input_type -> fsutil.types.Packet
	0, // 2: moby.filesync.v1.FileSend.DiffCopy:input_type -> moby.filesync.v1.BytesMessage
	1, // 3: moby.filesync.v1.FileSync.DiffCopy:output_type -> fsutil.types.Packet
	1, // 4: moby.filesync.v1.FileSync.TarStream:output_type -> fsutil.types.Packet
	0, // 5: moby.filesync.v1.FileSend.DiffCopy:output_type -> moby.filesync.v1.BytesMessage
	3, // [3:6] is the sub-list for method output_type
	0, // [0:3] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_github_com_moby_buildkit_session_filesync_filesync_proto_init() }
func file_github_com_moby_buildkit_session_filesync_filesync_proto_init() {
	if File_github_com_moby_buildkit_session_filesync_filesync_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_github_com_moby_buildkit_session_filesync_filesync_proto_rawDesc), len(file_github_com_moby_buildkit_session_filesync_filesync_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   2,
		},
		GoTypes:           file_github_com_moby_buildkit_session_filesync_filesync_proto_goTypes,
		DependencyIndexes: file_github_com_moby_buildkit_session_filesync_filesync_proto_depIdxs,
		MessageInfos:      file_github_com_moby_buildkit_session_filesync_filesync_proto_msgTypes,
	}.Build()
	File_github_com_moby_buildkit_session_filesync_filesync_proto = out.File
	file_github_com_moby_buildkit_session_filesync_filesync_proto_goTypes = nil
	file_github_com_moby_buildkit_session_filesync_filesync_proto_depIdxs = nil
}
