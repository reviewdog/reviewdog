// Reviewdog Diagnostic Format
//
// Reviewdog Diagnostic Format defines generic machine readable message
// structures which represents a result of diagnostic tool such as a compiler
// or a linter.
//
// The idea behind the Reviewdog Diagnostic Format is to standardize
// the protocol for how diagnostic tools (e.g. compilers, linters, etc..) and
// development tools (e.g. editors, reviewdog, etc..) communicate.
//
// Wire formats of Reviewdog Diagnostic Format.
// - rdjsonl: JSON Lines (http://jsonlines.org/) of the `Diagnostic` message.
// - rdjson: JSON format of the `DiagnosticResult` message.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.4
// 	protoc        v5.29.3
// source: reviewdog.proto

package rdf

import (
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

type Severity int32

const (
	Severity_UNKNOWN_SEVERITY Severity = 0
	Severity_ERROR            Severity = 1
	Severity_WARNING          Severity = 2
	Severity_INFO             Severity = 3
)

// Enum value maps for Severity.
var (
	Severity_name = map[int32]string{
		0: "UNKNOWN_SEVERITY",
		1: "ERROR",
		2: "WARNING",
		3: "INFO",
	}
	Severity_value = map[string]int32{
		"UNKNOWN_SEVERITY": 0,
		"ERROR":            1,
		"WARNING":          2,
		"INFO":             3,
	}
)

func (x Severity) Enum() *Severity {
	p := new(Severity)
	*p = x
	return p
}

func (x Severity) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Severity) Descriptor() protoreflect.EnumDescriptor {
	return file_reviewdog_proto_enumTypes[0].Descriptor()
}

func (Severity) Type() protoreflect.EnumType {
	return &file_reviewdog_proto_enumTypes[0]
}

func (x Severity) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Severity.Descriptor instead.
func (Severity) EnumDescriptor() ([]byte, []int) {
	return file_reviewdog_proto_rawDescGZIP(), []int{0}
}

// Result of diagnostic tool such as a compiler or a linter.
// It's intended to be used as top-level structured format which represents a
// whole result of a diagnostic tool.
type DiagnosticResult struct {
	state       protoimpl.MessageState `protogen:"open.v1"`
	Diagnostics []*Diagnostic          `protobuf:"bytes,1,rep,name=diagnostics,proto3" json:"diagnostics,omitempty"`
	// The source of diagnostics, e.g. 'typescript' or 'super lint'.
	// Optional.
	Source *Source `protobuf:"bytes,2,opt,name=source,proto3" json:"source,omitempty"`
	// This diagnostics' overall severity.
	// Optional.
	Severity      Severity `protobuf:"varint,3,opt,name=severity,proto3,enum=reviewdog.rdf.Severity" json:"severity,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DiagnosticResult) Reset() {
	*x = DiagnosticResult{}
	mi := &file_reviewdog_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DiagnosticResult) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DiagnosticResult) ProtoMessage() {}

func (x *DiagnosticResult) ProtoReflect() protoreflect.Message {
	mi := &file_reviewdog_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DiagnosticResult.ProtoReflect.Descriptor instead.
func (*DiagnosticResult) Descriptor() ([]byte, []int) {
	return file_reviewdog_proto_rawDescGZIP(), []int{0}
}

func (x *DiagnosticResult) GetDiagnostics() []*Diagnostic {
	if x != nil {
		return x.Diagnostics
	}
	return nil
}

func (x *DiagnosticResult) GetSource() *Source {
	if x != nil {
		return x.Source
	}
	return nil
}

func (x *DiagnosticResult) GetSeverity() Severity {
	if x != nil {
		return x.Severity
	}
	return Severity_UNKNOWN_SEVERITY
}

// Represents a diagnostic, such as a compiler error or warning.
// It's intended to be used as structured format which represents a
// diagnostic and can be used as stream of input/output such as jsonl.
// This message should be self-contained to report a diagnostic.
type Diagnostic struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// The diagnostic's message.
	Message string `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	// Location at which this diagnostic message applies.
	Location *Location `protobuf:"bytes,2,opt,name=location,proto3" json:"location,omitempty"`
	// This diagnostic's severity.
	// Optional.
	Severity Severity `protobuf:"varint,3,opt,name=severity,proto3,enum=reviewdog.rdf.Severity" json:"severity,omitempty"`
	// The source of this diagnostic, e.g. 'typescript' or 'super lint'.
	// Optional.
	Source *Source `protobuf:"bytes,4,opt,name=source,proto3" json:"source,omitempty"`
	// This diagnostic's rule code.
	// Optional.
	Code *Code `protobuf:"bytes,5,opt,name=code,proto3" json:"code,omitempty"`
	// Suggested fixes to resolve this diagnostic.
	// Optional.
	Suggestions []*Suggestion `protobuf:"bytes,6,rep,name=suggestions,proto3" json:"suggestions,omitempty"`
	// Experimental: If this diagnostic is converted from other formats,
	// original_output represents the original output which corresponds to this
	// diagnostic.
	// Optional.
	OriginalOutput string `protobuf:"bytes,7,opt,name=original_output,json=originalOutput,proto3" json:"original_output,omitempty"`
	// Related locations for this diagnostic.
	// Optional.
	RelatedLocations []*RelatedLocation `protobuf:"bytes,8,rep,name=related_locations,json=relatedLocations,proto3" json:"related_locations,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *Diagnostic) Reset() {
	*x = Diagnostic{}
	mi := &file_reviewdog_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Diagnostic) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Diagnostic) ProtoMessage() {}

func (x *Diagnostic) ProtoReflect() protoreflect.Message {
	mi := &file_reviewdog_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Diagnostic.ProtoReflect.Descriptor instead.
func (*Diagnostic) Descriptor() ([]byte, []int) {
	return file_reviewdog_proto_rawDescGZIP(), []int{1}
}

func (x *Diagnostic) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *Diagnostic) GetLocation() *Location {
	if x != nil {
		return x.Location
	}
	return nil
}

func (x *Diagnostic) GetSeverity() Severity {
	if x != nil {
		return x.Severity
	}
	return Severity_UNKNOWN_SEVERITY
}

func (x *Diagnostic) GetSource() *Source {
	if x != nil {
		return x.Source
	}
	return nil
}

func (x *Diagnostic) GetCode() *Code {
	if x != nil {
		return x.Code
	}
	return nil
}

func (x *Diagnostic) GetSuggestions() []*Suggestion {
	if x != nil {
		return x.Suggestions
	}
	return nil
}

func (x *Diagnostic) GetOriginalOutput() string {
	if x != nil {
		return x.OriginalOutput
	}
	return ""
}

func (x *Diagnostic) GetRelatedLocations() []*RelatedLocation {
	if x != nil {
		return x.RelatedLocations
	}
	return nil
}

type Location struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// File path. It could be either absolute path or relative path.
	Path string `protobuf:"bytes,2,opt,name=path,proto3" json:"path,omitempty"`
	// Range in the file path.
	// Optional.
	Range         *Range `protobuf:"bytes,3,opt,name=range,proto3" json:"range,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location) Reset() {
	*x = Location{}
	mi := &file_reviewdog_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location) ProtoMessage() {}

func (x *Location) ProtoReflect() protoreflect.Message {
	mi := &file_reviewdog_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location.ProtoReflect.Descriptor instead.
func (*Location) Descriptor() ([]byte, []int) {
	return file_reviewdog_proto_rawDescGZIP(), []int{2}
}

func (x *Location) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

func (x *Location) GetRange() *Range {
	if x != nil {
		return x.Range
	}
	return nil
}

type RelatedLocation struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Explanation of this related location.
	// Optional.
	Message string `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	// Required.
	Location      *Location `protobuf:"bytes,2,opt,name=location,proto3" json:"location,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RelatedLocation) Reset() {
	*x = RelatedLocation{}
	mi := &file_reviewdog_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RelatedLocation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RelatedLocation) ProtoMessage() {}

func (x *RelatedLocation) ProtoReflect() protoreflect.Message {
	mi := &file_reviewdog_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RelatedLocation.ProtoReflect.Descriptor instead.
func (*RelatedLocation) Descriptor() ([]byte, []int) {
	return file_reviewdog_proto_rawDescGZIP(), []int{3}
}

func (x *RelatedLocation) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *RelatedLocation) GetLocation() *Location {
	if x != nil {
		return x.Location
	}
	return nil
}

// start: { line: 2, column: 1 }
// end:   { line: 2, column: 4 }
//
//	=> "abc" (without line-break)
type Range struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Required.
	Start *Position `protobuf:"bytes,1,opt,name=start,proto3" json:"start,omitempty"`
	// end can be omitted. Then the range is handled as zero-length (start == end).
	// Optional.
	End           *Position `protobuf:"bytes,2,opt,name=end,proto3" json:"end,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Range) Reset() {
	*x = Range{}
	mi := &file_reviewdog_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Range) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Range) ProtoMessage() {}

func (x *Range) ProtoReflect() protoreflect.Message {
	mi := &file_reviewdog_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Range.ProtoReflect.Descriptor instead.
func (*Range) Descriptor() ([]byte, []int) {
	return file_reviewdog_proto_rawDescGZIP(), []int{4}
}

func (x *Range) GetStart() *Position {
	if x != nil {
		return x.Start
	}
	return nil
}

func (x *Range) GetEnd() *Position {
	if x != nil {
		return x.End
	}
	return nil
}

type Position struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Line number, starting at 1.
	// Optional.
	Line int32 `protobuf:"varint,1,opt,name=line,proto3" json:"line,omitempty"`
	// Column number, starting at 1 (byte count in UTF-8).
	// Example: 'a𐐀b'
	//
	//	The column of a: 1
	//	The column of 𐐀: 2
	//	The column of b: 6 since 𐐀 is represented with 4 bytes in UTF-8.
	//
	// Optional.
	Column        int32 `protobuf:"varint,2,opt,name=column,proto3" json:"column,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Position) Reset() {
	*x = Position{}
	mi := &file_reviewdog_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Position) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Position) ProtoMessage() {}

func (x *Position) ProtoReflect() protoreflect.Message {
	mi := &file_reviewdog_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Position.ProtoReflect.Descriptor instead.
func (*Position) Descriptor() ([]byte, []int) {
	return file_reviewdog_proto_rawDescGZIP(), []int{5}
}

func (x *Position) GetLine() int32 {
	if x != nil {
		return x.Line
	}
	return 0
}

func (x *Position) GetColumn() int32 {
	if x != nil {
		return x.Column
	}
	return 0
}

// Suggestion represents a suggested text manipulation to resolve a diagnostic
// problem.
//
// Insert example ('hayabusa' -> 'haya15busa'):
//
//	range {
//	  start {
//	    line: 1
//	    column: 5
//	  }
//	  end {
//	    line: 1
//	    column: 5
//	  }
//	}
//	text: 15
//
// |h|a|y|a|b|u|s|a|
// 1 2 3 4 5 6 7 8 9
//
//	^--- insert '15'
//
// Update example ('haya15busa' -> 'haya14busa'):
//
//	range {
//	  start {
//	    line: 1
//	    column: 5
//	  }
//	  end {
//	    line: 1
//	    column: 7
//	  }
//	}
//	text: 14
//
// |h|a|y|a|1|5|b|u|s|a|
// 1 2 3 4 5 6 7 8 9 0 1
//
//	^---^ replace with '14'
type Suggestion struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Range at which this suggestion applies.
	// To insert text into a document create a range where start == end.
	Range *Range `protobuf:"bytes,1,opt,name=range,proto3" json:"range,omitempty"`
	// A suggested text which replace the range.
	// For delete operations use an empty string.
	Text          string `protobuf:"bytes,2,opt,name=text,proto3" json:"text,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Suggestion) Reset() {
	*x = Suggestion{}
	mi := &file_reviewdog_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Suggestion) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Suggestion) ProtoMessage() {}

func (x *Suggestion) ProtoReflect() protoreflect.Message {
	mi := &file_reviewdog_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Suggestion.ProtoReflect.Descriptor instead.
func (*Suggestion) Descriptor() ([]byte, []int) {
	return file_reviewdog_proto_rawDescGZIP(), []int{6}
}

func (x *Suggestion) GetRange() *Range {
	if x != nil {
		return x.Range
	}
	return nil
}

func (x *Suggestion) GetText() string {
	if x != nil {
		return x.Text
	}
	return ""
}

type Source struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// A human-readable string describing the source of diagnostics, e.g.
	// 'typescript' or 'super lint'.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// URL to this source.
	// Optional.
	Url           string `protobuf:"bytes,2,opt,name=url,proto3" json:"url,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Source) Reset() {
	*x = Source{}
	mi := &file_reviewdog_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Source) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Source) ProtoMessage() {}

func (x *Source) ProtoReflect() protoreflect.Message {
	mi := &file_reviewdog_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Source.ProtoReflect.Descriptor instead.
func (*Source) Descriptor() ([]byte, []int) {
	return file_reviewdog_proto_rawDescGZIP(), []int{7}
}

func (x *Source) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Source) GetUrl() string {
	if x != nil {
		return x.Url
	}
	return ""
}

type Code struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// This rule's code/identifier.
	Value string `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	// A URL to open with more information about this rule code.
	// Optional.
	Url           string `protobuf:"bytes,2,opt,name=url,proto3" json:"url,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Code) Reset() {
	*x = Code{}
	mi := &file_reviewdog_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Code) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Code) ProtoMessage() {}

func (x *Code) ProtoReflect() protoreflect.Message {
	mi := &file_reviewdog_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Code.ProtoReflect.Descriptor instead.
func (*Code) Descriptor() ([]byte, []int) {
	return file_reviewdog_proto_rawDescGZIP(), []int{8}
}

func (x *Code) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

func (x *Code) GetUrl() string {
	if x != nil {
		return x.Url
	}
	return ""
}

var File_reviewdog_proto protoreflect.FileDescriptor

var file_reviewdog_proto_rawDesc = string([]byte{
	0x0a, 0x0f, 0x72, 0x65, 0x76, 0x69, 0x65, 0x77, 0x64, 0x6f, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x0d, 0x72, 0x65, 0x76, 0x69, 0x65, 0x77, 0x64, 0x6f, 0x67, 0x2e, 0x72, 0x64, 0x66,
	0x22, 0xb3, 0x01, 0x0a, 0x10, 0x44, 0x69, 0x61, 0x67, 0x6e, 0x6f, 0x73, 0x74, 0x69, 0x63, 0x52,
	0x65, 0x73, 0x75, 0x6c, 0x74, 0x12, 0x3b, 0x0a, 0x0b, 0x64, 0x69, 0x61, 0x67, 0x6e, 0x6f, 0x73,
	0x74, 0x69, 0x63, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x72, 0x65, 0x76,
	0x69, 0x65, 0x77, 0x64, 0x6f, 0x67, 0x2e, 0x72, 0x64, 0x66, 0x2e, 0x44, 0x69, 0x61, 0x67, 0x6e,
	0x6f, 0x73, 0x74, 0x69, 0x63, 0x52, 0x0b, 0x64, 0x69, 0x61, 0x67, 0x6e, 0x6f, 0x73, 0x74, 0x69,
	0x63, 0x73, 0x12, 0x2d, 0x0a, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x15, 0x2e, 0x72, 0x65, 0x76, 0x69, 0x65, 0x77, 0x64, 0x6f, 0x67, 0x2e, 0x72,
	0x64, 0x66, 0x2e, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x52, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x12, 0x33, 0x0a, 0x08, 0x73, 0x65, 0x76, 0x65, 0x72, 0x69, 0x74, 0x79, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x0e, 0x32, 0x17, 0x2e, 0x72, 0x65, 0x76, 0x69, 0x65, 0x77, 0x64, 0x6f, 0x67, 0x2e,
	0x72, 0x64, 0x66, 0x2e, 0x53, 0x65, 0x76, 0x65, 0x72, 0x69, 0x74, 0x79, 0x52, 0x08, 0x73, 0x65,
	0x76, 0x65, 0x72, 0x69, 0x74, 0x79, 0x22, 0x9b, 0x03, 0x0a, 0x0a, 0x44, 0x69, 0x61, 0x67, 0x6e,
	0x6f, 0x73, 0x74, 0x69, 0x63, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12,
	0x33, 0x0a, 0x08, 0x6c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x17, 0x2e, 0x72, 0x65, 0x76, 0x69, 0x65, 0x77, 0x64, 0x6f, 0x67, 0x2e, 0x72, 0x64,
	0x66, 0x2e, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x08, 0x6c, 0x6f, 0x63, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x12, 0x33, 0x0a, 0x08, 0x73, 0x65, 0x76, 0x65, 0x72, 0x69, 0x74, 0x79,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x17, 0x2e, 0x72, 0x65, 0x76, 0x69, 0x65, 0x77, 0x64,
	0x6f, 0x67, 0x2e, 0x72, 0x64, 0x66, 0x2e, 0x53, 0x65, 0x76, 0x65, 0x72, 0x69, 0x74, 0x79, 0x52,
	0x08, 0x73, 0x65, 0x76, 0x65, 0x72, 0x69, 0x74, 0x79, 0x12, 0x2d, 0x0a, 0x06, 0x73, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x72, 0x65, 0x76, 0x69,
	0x65, 0x77, 0x64, 0x6f, 0x67, 0x2e, 0x72, 0x64, 0x66, 0x2e, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x52, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x27, 0x0a, 0x04, 0x63, 0x6f, 0x64, 0x65,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x72, 0x65, 0x76, 0x69, 0x65, 0x77, 0x64,
	0x6f, 0x67, 0x2e, 0x72, 0x64, 0x66, 0x2e, 0x43, 0x6f, 0x64, 0x65, 0x52, 0x04, 0x63, 0x6f, 0x64,
	0x65, 0x12, 0x3b, 0x0a, 0x0b, 0x73, 0x75, 0x67, 0x67, 0x65, 0x73, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x18, 0x06, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x72, 0x65, 0x76, 0x69, 0x65, 0x77, 0x64,
	0x6f, 0x67, 0x2e, 0x72, 0x64, 0x66, 0x2e, 0x53, 0x75, 0x67, 0x67, 0x65, 0x73, 0x74, 0x69, 0x6f,
	0x6e, 0x52, 0x0b, 0x73, 0x75, 0x67, 0x67, 0x65, 0x73, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x27,
	0x0a, 0x0f, 0x6f, 0x72, 0x69, 0x67, 0x69, 0x6e, 0x61, 0x6c, 0x5f, 0x6f, 0x75, 0x74, 0x70, 0x75,
	0x74, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x6f, 0x72, 0x69, 0x67, 0x69, 0x6e, 0x61,
	0x6c, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x12, 0x4b, 0x0a, 0x11, 0x72, 0x65, 0x6c, 0x61, 0x74,
	0x65, 0x64, 0x5f, 0x6c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x08, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x72, 0x65, 0x76, 0x69, 0x65, 0x77, 0x64, 0x6f, 0x67, 0x2e, 0x72,
	0x64, 0x66, 0x2e, 0x52, 0x65, 0x6c, 0x61, 0x74, 0x65, 0x64, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x52, 0x10, 0x72, 0x65, 0x6c, 0x61, 0x74, 0x65, 0x64, 0x4c, 0x6f, 0x63, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x22, 0x4a, 0x0a, 0x08, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x12, 0x12, 0x0a, 0x04, 0x70, 0x61, 0x74, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x70, 0x61, 0x74, 0x68, 0x12, 0x2a, 0x0a, 0x05, 0x72, 0x61, 0x6e, 0x67, 0x65, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x72, 0x65, 0x76, 0x69, 0x65, 0x77, 0x64, 0x6f, 0x67, 0x2e,
	0x72, 0x64, 0x66, 0x2e, 0x52, 0x61, 0x6e, 0x67, 0x65, 0x52, 0x05, 0x72, 0x61, 0x6e, 0x67, 0x65,
	0x22, 0x60, 0x0a, 0x0f, 0x52, 0x65, 0x6c, 0x61, 0x74, 0x65, 0x64, 0x4c, 0x6f, 0x63, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x33, 0x0a,
	0x08, 0x6c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x17, 0x2e, 0x72, 0x65, 0x76, 0x69, 0x65, 0x77, 0x64, 0x6f, 0x67, 0x2e, 0x72, 0x64, 0x66, 0x2e,
	0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x08, 0x6c, 0x6f, 0x63, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x22, 0x61, 0x0a, 0x05, 0x52, 0x61, 0x6e, 0x67, 0x65, 0x12, 0x2d, 0x0a, 0x05, 0x73,
	0x74, 0x61, 0x72, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x72, 0x65, 0x76,
	0x69, 0x65, 0x77, 0x64, 0x6f, 0x67, 0x2e, 0x72, 0x64, 0x66, 0x2e, 0x50, 0x6f, 0x73, 0x69, 0x74,
	0x69, 0x6f, 0x6e, 0x52, 0x05, 0x73, 0x74, 0x61, 0x72, 0x74, 0x12, 0x29, 0x0a, 0x03, 0x65, 0x6e,
	0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x72, 0x65, 0x76, 0x69, 0x65, 0x77,
	0x64, 0x6f, 0x67, 0x2e, 0x72, 0x64, 0x66, 0x2e, 0x50, 0x6f, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e,
	0x52, 0x03, 0x65, 0x6e, 0x64, 0x22, 0x36, 0x0a, 0x08, 0x50, 0x6f, 0x73, 0x69, 0x74, 0x69, 0x6f,
	0x6e, 0x12, 0x12, 0x0a, 0x04, 0x6c, 0x69, 0x6e, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x04, 0x6c, 0x69, 0x6e, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x06, 0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x22, 0x4c, 0x0a,
	0x0a, 0x53, 0x75, 0x67, 0x67, 0x65, 0x73, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x2a, 0x0a, 0x05, 0x72,
	0x61, 0x6e, 0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x72, 0x65, 0x76,
	0x69, 0x65, 0x77, 0x64, 0x6f, 0x67, 0x2e, 0x72, 0x64, 0x66, 0x2e, 0x52, 0x61, 0x6e, 0x67, 0x65,
	0x52, 0x05, 0x72, 0x61, 0x6e, 0x67, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65, 0x78, 0x74, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x65, 0x78, 0x74, 0x22, 0x2e, 0x0a, 0x06, 0x53,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x72, 0x6c,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x75, 0x72, 0x6c, 0x22, 0x2e, 0x0a, 0x04, 0x43,
	0x6f, 0x64, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x72, 0x6c,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x75, 0x72, 0x6c, 0x2a, 0x42, 0x0a, 0x08, 0x53,
	0x65, 0x76, 0x65, 0x72, 0x69, 0x74, 0x79, 0x12, 0x14, 0x0a, 0x10, 0x55, 0x4e, 0x4b, 0x4e, 0x4f,
	0x57, 0x4e, 0x5f, 0x53, 0x45, 0x56, 0x45, 0x52, 0x49, 0x54, 0x59, 0x10, 0x00, 0x12, 0x09, 0x0a,
	0x05, 0x45, 0x52, 0x52, 0x4f, 0x52, 0x10, 0x01, 0x12, 0x0b, 0x0a, 0x07, 0x57, 0x41, 0x52, 0x4e,
	0x49, 0x4e, 0x47, 0x10, 0x02, 0x12, 0x08, 0x0a, 0x04, 0x49, 0x4e, 0x46, 0x4f, 0x10, 0x03, 0x42,
	0x2a, 0x5a, 0x28, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x72, 0x65,
	0x76, 0x69, 0x65, 0x77, 0x64, 0x6f, 0x67, 0x2f, 0x72, 0x65, 0x76, 0x69, 0x65, 0x77, 0x64, 0x6f,
	0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x72, 0x64, 0x66, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
})

var (
	file_reviewdog_proto_rawDescOnce sync.Once
	file_reviewdog_proto_rawDescData []byte
)

func file_reviewdog_proto_rawDescGZIP() []byte {
	file_reviewdog_proto_rawDescOnce.Do(func() {
		file_reviewdog_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_reviewdog_proto_rawDesc), len(file_reviewdog_proto_rawDesc)))
	})
	return file_reviewdog_proto_rawDescData
}

var file_reviewdog_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_reviewdog_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_reviewdog_proto_goTypes = []any{
	(Severity)(0),            // 0: reviewdog.rdf.Severity
	(*DiagnosticResult)(nil), // 1: reviewdog.rdf.DiagnosticResult
	(*Diagnostic)(nil),       // 2: reviewdog.rdf.Diagnostic
	(*Location)(nil),         // 3: reviewdog.rdf.Location
	(*RelatedLocation)(nil),  // 4: reviewdog.rdf.RelatedLocation
	(*Range)(nil),            // 5: reviewdog.rdf.Range
	(*Position)(nil),         // 6: reviewdog.rdf.Position
	(*Suggestion)(nil),       // 7: reviewdog.rdf.Suggestion
	(*Source)(nil),           // 8: reviewdog.rdf.Source
	(*Code)(nil),             // 9: reviewdog.rdf.Code
}
var file_reviewdog_proto_depIdxs = []int32{
	2,  // 0: reviewdog.rdf.DiagnosticResult.diagnostics:type_name -> reviewdog.rdf.Diagnostic
	8,  // 1: reviewdog.rdf.DiagnosticResult.source:type_name -> reviewdog.rdf.Source
	0,  // 2: reviewdog.rdf.DiagnosticResult.severity:type_name -> reviewdog.rdf.Severity
	3,  // 3: reviewdog.rdf.Diagnostic.location:type_name -> reviewdog.rdf.Location
	0,  // 4: reviewdog.rdf.Diagnostic.severity:type_name -> reviewdog.rdf.Severity
	8,  // 5: reviewdog.rdf.Diagnostic.source:type_name -> reviewdog.rdf.Source
	9,  // 6: reviewdog.rdf.Diagnostic.code:type_name -> reviewdog.rdf.Code
	7,  // 7: reviewdog.rdf.Diagnostic.suggestions:type_name -> reviewdog.rdf.Suggestion
	4,  // 8: reviewdog.rdf.Diagnostic.related_locations:type_name -> reviewdog.rdf.RelatedLocation
	5,  // 9: reviewdog.rdf.Location.range:type_name -> reviewdog.rdf.Range
	3,  // 10: reviewdog.rdf.RelatedLocation.location:type_name -> reviewdog.rdf.Location
	6,  // 11: reviewdog.rdf.Range.start:type_name -> reviewdog.rdf.Position
	6,  // 12: reviewdog.rdf.Range.end:type_name -> reviewdog.rdf.Position
	5,  // 13: reviewdog.rdf.Suggestion.range:type_name -> reviewdog.rdf.Range
	14, // [14:14] is the sub-list for method output_type
	14, // [14:14] is the sub-list for method input_type
	14, // [14:14] is the sub-list for extension type_name
	14, // [14:14] is the sub-list for extension extendee
	0,  // [0:14] is the sub-list for field type_name
}

func init() { file_reviewdog_proto_init() }
func file_reviewdog_proto_init() {
	if File_reviewdog_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_reviewdog_proto_rawDesc), len(file_reviewdog_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_reviewdog_proto_goTypes,
		DependencyIndexes: file_reviewdog_proto_depIdxs,
		EnumInfos:         file_reviewdog_proto_enumTypes,
		MessageInfos:      file_reviewdog_proto_msgTypes,
	}.Build()
	File_reviewdog_proto = out.File
	file_reviewdog_proto_goTypes = nil
	file_reviewdog_proto_depIdxs = nil
}
