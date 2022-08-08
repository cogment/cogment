// Copyright 2022 AI Redefined Inc. <dev+cogment@ai-r.com>
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

package client

import (
	"fmt"

	jsonEncoding "encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type consoleOutputFormat string

const (
	text consoleOutputFormat = "text"
	json consoleOutputFormat = "json"
)

var expectedOutputFormats = []consoleOutputFormat{text, json}

func retrieveConsoleOutputFormat() (consoleOutputFormat, error) {
	cfgOutputFormat := consoleOutputFormat(clientViper.GetString(clientConsoleOutputFormatKey))

	for _, format := range expectedOutputFormats {
		if format == cfgOutputFormat {
			return cfgOutputFormat, nil
		}
	}
	return consoleOutputFormat("invalid"), fmt.Errorf(
		"invalid output format specified %q expecting one of %v",
		cfgOutputFormat,
		expectedOutputFormats,
	)
}

func renderJSONFromProto(message protoreflect.ProtoMessage) error {
	jsonMarshaller := &protojson.MarshalOptions{
		Multiline:       false,
		Indent:          "",
		EmitUnpopulated: false,
		UseEnumNumbers:  false,
	}
	serializedMessage, err := jsonMarshaller.Marshal(message)
	if err != nil {
		return err
	}

	fmt.Println(string(serializedMessage))
	return nil
}

func renderJSON(message interface{}) error {
	serializedMessage, err := jsonEncoding.Marshal(message)
	if err != nil {
		return err
	}

	fmt.Println(string(serializedMessage))
	return nil
}
