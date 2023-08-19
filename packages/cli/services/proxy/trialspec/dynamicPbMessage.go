// Copyright 2023 AI Redefined Inc. <dev+cogment@ai-r.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package trialspec

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type DynamicPbMessage dynamicpb.Message

func NewMessageFromDescriptor(descriptor protoreflect.MessageDescriptor) *DynamicPbMessage {
	return (*DynamicPbMessage)(dynamicpb.NewMessage(descriptor))
}

func (dpm *DynamicPbMessage) MarshalJSON() ([]byte, error) {
	marshal := protojson.MarshalOptions{
		UseProtoNames:   true, // use the snake_case proto names, as it is Cogment's convention for json anyway.
		EmitUnpopulated: true,
		UseEnumNumbers:  false,
	}
	return marshal.Marshal((*dynamicpb.Message)(dpm))
}

func (dpm *DynamicPbMessage) UnmarshalJSON(b []byte) (err error) {
	return protojson.Unmarshal(b, (*dynamicpb.Message)(dpm))
}

func (dpm *DynamicPbMessage) MarshalProto() ([]byte, error) {
	return proto.Marshal((*dynamicpb.Message)(dpm))
}

func (dpm *DynamicPbMessage) UnmarshalProto(b []byte) (err error) {
	return proto.Unmarshal(b, (*dynamicpb.Message)(dpm))
}

func (dpm *DynamicPbMessage) String() string {
	return (*dynamicpb.Message)(dpm).String()
}

func (dpm *DynamicPbMessage) IsValid() bool {
	return (*dynamicpb.Message)(dpm).IsValid()
}

func (dpm *DynamicPbMessage) ProtoReflect() protoreflect.Message {
	return (*dynamicpb.Message)(dpm).ProtoReflect()
}
