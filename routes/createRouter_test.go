// Copyright 2021 AI Redefined Inc. <dev+cogment@ai-r.com>
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

//nolint:staticcheck
package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/cogment/model-registry/backend"
	"github.com/cogment/model-registry/backend/db"
	"github.com/stretchr/testify/assert"
)

var modelData = []byte(`Lorem ipsum dolor sit amet, consectetuer adipiscing elit. Aenean commodo ligula
eget dolor. Aenean massa. Cum sociis natoque penatibus et magnis dis parturient
montes, nascetur ridiculus mus. Donec quam felis, ultricies nec, pellentesque
eu, pretium quis, sem. Nulla consequat massa quis enim. Donec pede justo,
fringilla vel, aliquet nec, vulputate eget, arcu. In enim justo, rhoncus ut,
imperdiet a, venenatis vitae, justo. Nullam dictum felis eu pede mollis pretium.
Integer tincidunt. Cras dapibus. Vivamus elementum semper nisi. Aenean vulputate
eleifend tellus. Aenean leo ligula, porttitor eu, consequat vitae, eleifend ac,
enim. Aliquam lorem ante, dapibus in, viverra quis, feugiat a, tellus. Phasellus
viverra nulla ut metus varius laoreet.`)

func recordResponse(t *testing.T, b backend.Backend, method string, route string, body io.Reader) *httptest.ResponseRecorder {
	req, err := http.NewRequest(method, route, body)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	handler := CreateRouter(b)

	handler.ServeHTTP(rr, req)

	return rr
}

func TestWelcome(t *testing.T) {
	b, err := db.CreateBackend()
	assert.NoError(t, err)
	defer b.Destroy()

	rr := recordResponse(t, b, "GET", "/", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Equal(t, "{\"message\":\"Welcome to the model registry API\"}\n", rr.Body.String())
}

func Test404(t *testing.T) {
	b, err := db.CreateBackend()
	assert.NoError(t, err)
	defer b.Destroy()

	rr := recordResponse(t, b, "GET", "/foo", nil)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Equal(t, "{\"status\":404,\"message\":\"/foo not found\"}\n", rr.Body.String())
}

func TestListModel(t *testing.T) {
	b, err := db.CreateBackend()
	assert.NoError(t, err)
	defer b.Destroy()

	rr := recordResponse(t, b, "GET", "/models", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Equal(t, "[]\n", rr.Body.String())

	err = b.CreateModel("foo")
	assert.NoError(t, err)

	_, err = b.CreateOrUpdateModelVersion("foo", -1, modelData, true)
	assert.NoError(t, err)

	_, err = b.CreateOrUpdateModelVersion("foo", -1, modelData, false)
	assert.NoError(t, err)

	err = b.CreateModel("bar")
	assert.NoError(t, err)

	err = b.CreateModel("baz")
	assert.NoError(t, err)

	_, err = b.CreateOrUpdateModelVersion("baz", -1, modelData, true)
	assert.NoError(t, err)

	rr = recordResponse(t, b, "GET", "/models", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Regexp(t, regexp.MustCompile("{\"modelId\":\"bar\",\"latestVersionCreatedAt\":\"[0-9-:TZ\\.]+\"},"+
		"{\"modelId\":\"baz\",\"latestVersionCreatedAt\":\"[0-9-:TZ\\.]+\",\"latestVersionNumber\":1},"+
		"[{\"modelId\":\"foo\",\"latestVersionCreatedAt\":\"[0-9-:TZ\\.]+\",\"latestVersionNumber\":2}]"), rr.Body.String())
}

func TestListModelVersions(t *testing.T) {
	b, err := db.CreateBackend()
	assert.NoError(t, err)
	defer b.Destroy()

	err = b.CreateModel("foo")
	assert.NoError(t, err)

	rr := recordResponse(t, b, "GET", "/models/foo", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Equal(t, "[]\n", rr.Body.String())

	for i := 0; i < 20; i++ {
		_, err := b.CreateOrUpdateModelVersion("foo", -1, modelData, false)
		assert.NoError(t, err)
	}

	rr = recordResponse(t, b, "GET", "/models/foo", nil)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])

	versions := []backend.VersionInfo{}
	err = json.NewDecoder(rr.Body).Decode(&versions)
	assert.NoError(t, err)
	assert.Len(t, versions, 20)
	for i := 0; i < 20; i++ {
		assert.Equal(t, i+1, versions[i].Number)
		assert.NotEmpty(t, versions[i].Hash)
		assert.Equal(t, "foo", versions[i].ModelID)
		assert.False(t, versions[i].Archive)
	}
}

func TestGetModelLatestVersion(t *testing.T) {
	b, err := db.CreateBackend()
	assert.NoError(t, err)
	defer b.Destroy()

	rr := recordResponse(t, b, "GET", "/models/foo/latest", nil)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Equal(t, "{\"status\":400,\"message\":\"Unable to retrieve the latest version of model \\\"foo\\\": model \\\"foo\\\" doesn't have any version yet\"}\n", rr.Body.String())

	err = b.CreateModel("foo")
	assert.NoError(t, err)

	rr = recordResponse(t, b, "GET", "/models/foo/latest", nil)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Equal(t, "{\"status\":400,\"message\":\"Unable to retrieve the latest version of model \\\"foo\\\": model \\\"foo\\\" doesn't have any version yet\"}\n", rr.Body.String())

	_, err = b.CreateOrUpdateModelVersion("foo", -1, modelData, false)
	assert.NoError(t, err)

	rr = recordResponse(t, b, "GET", "/models/foo/latest", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, []string{"application/octet-stream"}, rr.HeaderMap["Content-Type"])
	assert.Equal(t, modelData, rr.Body.Bytes())
}

func TestGetModelVersion(t *testing.T) {
	b, err := db.CreateBackend()
	assert.NoError(t, err)
	defer b.Destroy()

	rr := recordResponse(t, b, "GET", "/models/foo/12", nil)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Equal(t, "{\"status\":400,\"message\":\"Unable to retrieve the version \\\"12\\\" of model \\\"foo\\\": no version \\\"12\\\" for model \\\"foo\\\" found\"}\n", rr.Body.String())

	err = b.CreateModel("foo")
	assert.NoError(t, err)

	rr = recordResponse(t, b, "GET", "/models/foo/12", nil)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Equal(t, "{\"status\":400,\"message\":\"Unable to retrieve the version \\\"12\\\" of model \\\"foo\\\": no version \\\"12\\\" for model \\\"foo\\\" found\"}\n", rr.Body.String())

	for i := 0; i < 15; i++ {
		_, err = b.CreateOrUpdateModelVersion("foo", -1, []byte(fmt.Sprintf("%d", i)), false)
		assert.NoError(t, err)
	}

	rr = recordResponse(t, b, "GET", "/models/foo/12", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, []string{"application/octet-stream"}, rr.HeaderMap["Content-Type"])
	assert.Equal(t, []byte("11"), rr.Body.Bytes())
}

func TestAddModel(t *testing.T) {
	b, err := db.CreateBackend()
	assert.NoError(t, err)
	defer b.Destroy()

	rr := recordResponse(t, b, "POST", "/models", nil)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Equal(t, "{\"status\":400,\"message\":\"Missing request body\"}\n", rr.Body.String())

	body, _ := json.Marshal(map[string]string{"modelId": "my-cool-model"})
	rr = recordResponse(t, b, "POST", "/models", bytes.NewReader(body))

	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Regexp(t, regexp.MustCompile("{\"modelId\":\"my-cool-model\",\"latestVersionCreatedAt\":\"[0-9-:TZ\\.]+\"}"), rr.Body.String())

	rr = recordResponse(t, b, "POST", "/models", bytes.NewReader(body))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Equal(t, "{\"status\":400,\"message\":\"Unable to create model \\\"my-cool-model\\\": model \\\"my-cool-model\\\" already exists\"}\n", rr.Body.String())

	badBody, _ := json.Marshal(map[string]string{"foo": "my-cool-model"})
	rr = recordResponse(t, b, "POST", "/models", bytes.NewReader(badBody))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Equal(t, "{\"status\":400,\"message\":\"Request body contains unknown field \\\"foo\\\"\"}\n", rr.Body.String())
}

func TestAddModelVersion(t *testing.T) {
	b, err := db.CreateBackend()
	assert.NoError(t, err)
	defer b.Destroy()

	rr := recordResponse(t, b, "POST", "/models/foo", nil)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Equal(t, "{\"status\":400,\"message\":\"Missing request body\"}\n", rr.Body.String())

	rr = recordResponse(t, b, "POST", "/models/foo", bytes.NewReader(modelData))

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Equal(t, "{\"status\":400,\"message\":\"Unable to create a version for model \\\"foo\\\": no model \\\"foo\\\" found\"}\n", rr.Body.String())

	body, _ := json.Marshal(map[string]string{"modelId": "foo"})
	rr = recordResponse(t, b, "POST", "/models", bytes.NewReader(body))
	assert.Equal(t, http.StatusCreated, rr.Code)

	rr = recordResponse(t, b, "POST", "/models/foo", bytes.NewReader(modelData))

	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Regexp(t, regexp.MustCompile("{\"modelId\":\"foo\",\"createdAt\":\"[0-9-:TZ\\.]+\",\"number\":1,\"archive\":false,\"hash\":\"[^\\s=]{43}=\"}"), rr.Body.String())

	rr = recordResponse(t, b, "POST", "/models/foo?archive=true", bytes.NewReader(modelData))

	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Equal(t, []string{"application/json"}, rr.HeaderMap["Content-Type"])
	assert.Regexp(t, regexp.MustCompile("{\"modelId\":\"foo\",\"createdAt\":\"[0-9-:TZ\\.]+\",\"number\":2,\"archive\":true,\"hash\":\"[^\\s=]{43}=\"}"), rr.Body.String())
}
