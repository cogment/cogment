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

package endpoint

import (
	"encoding/json"
	"testing"

	grpcapi "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/stretchr/testify/assert"
)

func TestDefaultEndpoint(t *testing.T) {
	endpoint := Endpoint{}
	assert.False(t, endpoint.IsValid())
	assert.Equal(t, "", endpoint.String())
}

func TestParse(t *testing.T) {
	endpoint, err := Parse("cogment://discover/actor?__actor_class=foo")
	assert.NoError(t, err)
	assert.Equal(t, DiscoveryEndpoint, endpoint.Type())
	assert.Equal(t, grpcapi.ServiceType_ACTOR_SERVICE, endpoint.ServiceType())
	assert.Contains(t, endpoint.Properties(), ActorClassPropertyName)
	assert.Equal(t, "foo", endpoint.Properties()[ActorClassPropertyName])
}

type TestStruct struct {
	Endpoint Endpoint `json:"endpoint"`
	Foo      int      `json:"foo"`
}

func TestUnmarshalJSON(t *testing.T) {
	test := TestStruct{}
	err := json.Unmarshal([]byte("{\"endpoint\":\"cogment://discover?__id=12345\",\"foo\":42}"), &test)
	assert.NoError(t, err)
	assert.Equal(t, 42, test.Foo)
	assert.Equal(t, DiscoveryEndpoint, test.Endpoint.Type())
	assert.Equal(t, grpcapi.ServiceType_UNKNOWN_SERVICE, test.Endpoint.ServiceType())
	serviceID, hasServiceID := test.Endpoint.ServiceID()
	assert.True(t, hasServiceID)
	assert.Equal(t, 12345, int(serviceID))
}

func TestMarshalJSON(t *testing.T) {
	endpoint, err := Parse("grpc://192.168.0.1:9000")
	assert.NoError(t, err)
	test := TestStruct{
		Endpoint: endpoint,
		Foo:      666,
	}
	b, err := json.Marshal(test)
	assert.NoError(t, err)
	assert.Equal(t, "{\"endpoint\":\"grpc://192.168.0.1:9000\",\"foo\":666}", string(b))
}

type TestStructPtr struct {
	Endpoint *Endpoint `json:"endpoint"`
}

func TestMarshalJSONPtr(t *testing.T) {
	endpoint, err := Parse("cogment://client")
	assert.NoError(t, err)
	test := TestStructPtr{
		Endpoint: &endpoint,
	}
	b, err := json.Marshal(test)
	assert.NoError(t, err)
	assert.Equal(t, "{\"endpoint\":\"cogment://client\"}", string(b))
}
