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

package httpserver

import (
	"encoding/json"
	"reflect"

	"github.com/cogment/cogment/services/proxy/trialspec"
	"github.com/cogment/cogment/utils/endpoint"
	"github.com/wI2L/fizz/openapi"
)

// This file omplements the openapi.Typer interface on the `httpserver` types to make the OpenAPI spec looks better

func (*httpError) TypeName() string          { return "ErrorResponse" }
func (*infoResponse) TypeName() string       { return "APIInfoResponse" }
func (*startTrialResponse) TypeName() string { return "StartTrialResponse" }

func overrideTypes(generator *openapi.Generator) error {
	err := generator.OverrideDataType(reflect.TypeOf(json.RawMessage{}), "object", "")
	if err != nil {
		return err
	}
	err = generator.OverrideDataType(reflect.TypeOf(endpoint.Endpoint{}), "string", "")
	if err != nil {
		return err
	}
	err = generator.OverrideDataType(reflect.TypeOf(trialspec.DynamicPbMessage{}), "object", "")
	if err != nil {
		return err
	}
	return nil
}