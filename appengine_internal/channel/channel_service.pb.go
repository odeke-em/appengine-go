// Code generated by protoc-gen-go from "channel_service.proto"
// DO NOT EDIT!

package appengine

import proto "code.google.com/p/goprotobuf/proto"
import "math"

// Reference proto and math imports to suppress error if they are not otherwise used.
var _ = proto.GetString
var _ = math.Inf

type ChannelServiceError_ErrorCode int32

const (
	ChannelServiceError_OK                             ChannelServiceError_ErrorCode = 0
	ChannelServiceError_INTERNAL_ERROR                 ChannelServiceError_ErrorCode = 1
	ChannelServiceError_INVALID_CHANNEL_KEY            ChannelServiceError_ErrorCode = 2
	ChannelServiceError_BAD_MESSAGE                    ChannelServiceError_ErrorCode = 3
	ChannelServiceError_INVALID_CHANNEL_TOKEN_DURATION ChannelServiceError_ErrorCode = 4
	ChannelServiceError_APPID_ALIAS_REQUIRED           ChannelServiceError_ErrorCode = 5
)

var ChannelServiceError_ErrorCode_name = map[int32]string{
	0: "OK",
	1: "INTERNAL_ERROR",
	2: "INVALID_CHANNEL_KEY",
	3: "BAD_MESSAGE",
	4: "INVALID_CHANNEL_TOKEN_DURATION",
	5: "APPID_ALIAS_REQUIRED",
}
var ChannelServiceError_ErrorCode_value = map[string]int32{
	"OK":                             0,
	"INTERNAL_ERROR":                 1,
	"INVALID_CHANNEL_KEY":            2,
	"BAD_MESSAGE":                    3,
	"INVALID_CHANNEL_TOKEN_DURATION": 4,
	"APPID_ALIAS_REQUIRED":           5,
}

func NewChannelServiceError_ErrorCode(x ChannelServiceError_ErrorCode) *ChannelServiceError_ErrorCode {
	e := ChannelServiceError_ErrorCode(x)
	return &e
}
func (x ChannelServiceError_ErrorCode) String() string {
	return proto.EnumName(ChannelServiceError_ErrorCode_name, int32(x))
}

type ChannelServiceError struct {
	XXX_unrecognized []byte `json:"-"`
}

func (this *ChannelServiceError) Reset()         { *this = ChannelServiceError{} }
func (this *ChannelServiceError) String() string { return proto.CompactTextString(this) }

type CreateChannelRequest struct {
	ApplicationKey   *string `protobuf:"bytes,1,req,name=application_key" json:"application_key,omitempty"`
	DurationMinutes  *int32  `protobuf:"varint,2,opt,name=duration_minutes" json:"duration_minutes,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (this *CreateChannelRequest) Reset()         { *this = CreateChannelRequest{} }
func (this *CreateChannelRequest) String() string { return proto.CompactTextString(this) }

type CreateChannelResponse struct {
	Token            *string `protobuf:"bytes,2,opt,name=token" json:"token,omitempty"`
	DurationMinutes  *int32  `protobuf:"varint,3,opt,name=duration_minutes" json:"duration_minutes,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (this *CreateChannelResponse) Reset()         { *this = CreateChannelResponse{} }
func (this *CreateChannelResponse) String() string { return proto.CompactTextString(this) }

type SendMessageRequest struct {
	ApplicationKey   *string `protobuf:"bytes,1,req,name=application_key" json:"application_key,omitempty"`
	Message          *string `protobuf:"bytes,2,req,name=message" json:"message,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (this *SendMessageRequest) Reset()         { *this = SendMessageRequest{} }
func (this *SendMessageRequest) String() string { return proto.CompactTextString(this) }

func init() {
	proto.RegisterEnum("appengine.ChannelServiceError_ErrorCode", ChannelServiceError_ErrorCode_name, ChannelServiceError_ErrorCode_value)
}
