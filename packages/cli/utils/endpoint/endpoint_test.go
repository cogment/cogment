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

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/stretchr/testify/assert"
)

func TestDefaultEndpoint(t *testing.T) {
	endpoint := Endpoint{}
	assert.False(t, endpoint.IsValid())
	assert.Equal(t, "", endpoint.String())
}

func TestParseEmptyEndpoint(t *testing.T) {
	endpoint, err := Parse("")
	assert.Error(t, err)
	assert.False(t, endpoint.IsValid())
}

func TestParseBadEndpoint(t *testing.T) {
	endpoint, err := Parse("this is not a URL")
	assert.Error(t, err)
	assert.False(t, endpoint.IsValid())
}

func TestParseGrpcEndpoint(t *testing.T) {
	endpoint, err := Parse("grpc://localhost:46637")
	assert.NoError(t, err)
	assert.Equal(t, GrpcEndpoint, endpoint.Category)

	url, err := endpoint.ResolvedURL()
	assert.NoError(t, err)
	assert.Equal(t, "localhost:46637", url.Host)
}

func TestParseBadGrpcEndpoint(t *testing.T) {
	endpoint, err := Parse("grpc://localhost:46637/path/to/something")
	assert.Error(t, err)
	assert.False(t, endpoint.IsValid())
}

func TestParseClientEndpoint(t *testing.T) {
	endpoint, err := Parse("cogment://client")
	assert.NoError(t, err)
	assert.Equal(t, ClientEndpoint, endpoint.Category)

	url, err := endpoint.ResolvedURL()
	assert.Error(t, err)
	assert.Nil(t, url)
}

func TestParseDiscoveryEndpoint(t *testing.T) {
	endpoint, err := Parse("cogment://discover/actor?__actor_class=foo")
	assert.NoError(t, err)
	assert.Equal(t, DiscoveryEndpoint, endpoint.Category)
	assert.Equal(t, cogmentAPI.ServiceType_ACTOR_SERVICE, endpoint.Details.Type)
	assert.Contains(t, endpoint.Details.Properties, ActorClassPropertyName)
	assert.Equal(t, "foo", endpoint.Details.Properties[ActorClassPropertyName])
	assert.Equal(t, uint64(0), endpoint.ServiceDiscoveryID)

	assert.Equal(t, "cogment://discover/actor?__actor_class=foo", endpoint.String())

	assert.Equal(t, uint32(0), endpoint.Port())
	assert.Equal(t, "", endpoint.Host())

	url, err := endpoint.ResolvedURL()
	assert.Error(t, err)
	assert.Nil(t, url)
}

type TestStruct struct {
	Endpoint *Endpoint `json:"endpoint"`
	Foo      int       `json:"foo"`
}

func TestUnmarshalJSON(t *testing.T) {
	test := TestStruct{}
	err := json.Unmarshal([]byte("{\"endpoint\":\"cogment://discover/service?__id=12345\",\"foo\":42}"), &test)
	assert.NoError(t, err)
	assert.Equal(t, 42, test.Foo)
	assert.Equal(t, DiscoveryEndpoint, test.Endpoint.Category)
	assert.Equal(t, cogmentAPI.ServiceType_UNKNOWN_SERVICE, test.Endpoint.Details.Type)
	assert.Equal(t, uint64(12345), test.Endpoint.ServiceDiscoveryID)
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
