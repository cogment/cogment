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

package test

import (
	"testing"

	"github.com/cogment/model-registry/backend"
	"github.com/stretchr/testify/assert"
)

// Data1 is exemple model version data
var Data1 = []byte(`Lorem ipsum dolor sit amet, consectetuer adipiscing elit. Aenean commodo ligula
eget dolor. Aenean massa. Cum sociis natoque penatibus et magnis dis parturient
montes, nascetur ridiculus mus. Donec quam felis, ultricies nec, pellentesque
eu, pretium quis, sem. Nulla consequat massa quis enim. Donec pede justo,
fringilla vel, aliquet nec, vulputate eget, arcu. In enim justo, rhoncus ut,
imperdiet a, venenatis vitae, justo. Nullam dictum felis eu pede mollis pretium.
Integer tincidunt. Cras dapibus. Vivamus elementum semper nisi. Aenean vulputate
eleifend tellus. Aenean leo ligula, porttitor eu, consequat vitae, eleifend ac,
enim. Aliquam lorem ante, dapibus in, viverra quis, feugiat a, tellus. Phasellus
viverra nulla ut metus varius laoreet.`)

// Data2 is exemple model version data
var Data2 = []byte(`Quisque rutrum. Aenean imperdiet. Etiam
ultricies nisi vel augue. Curabitur ullamcorper ultricies nisi. Nam eget dui.
Lorem ipsum dolor sit amet, consectetuer adipiscing elit. Aenean commodo ligula
eget dolor. Aenean massa. Cum sociis natoque penatibus et magnis dis parturient
montes, nascetur ridiculus mus. Donec quam felis, ultricies nec, pellentesque
eu, pretium quis, sem. Nulla consequat massa quis enim. Donec pede justo,
fringilla vel, aliquet nec, vulputate eget, arcu. In enim justo, rhoncus ut,
imperdiet a, venenatis vitae, justo. Nullam dictum felis eu pede mollis pretium.
Integer tincidunt.`)

// RunSuite runs the full backend test suite
func RunSuite(t *testing.T, createBackend func() backend.Backend) {
	cases := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "TestCreateAndDestroyBackend",
			test: func(t *testing.T) {
				b := createBackend()
				assert.NotNil(t, b)
				b.Destroy()
			},
		},
		{
			name: "TestCreateModel",
			test: func(t *testing.T) {
				b := createBackend()
				defer b.Destroy()

				err := b.CreateModel("foo")
				assert.NoError(t, err)

				// Create a duplicate should fail
				err = b.CreateModel("foo")
				expectedErr := &backend.ModelAlreadyExistsError{}
				assert.ErrorAs(t, err, &expectedErr)
				assert.Equal(t, expectedErr.ModelID, "foo")
				assert.EqualError(t, err, "model \"foo\" already exists")

				// Create a another one should succeed
				err = b.CreateModel("bar")
				assert.NoError(t, err)
			},
		},
		{
			name: "TestHasModel",
			test: func(t *testing.T) {
				b := createBackend()
				defer b.Destroy()

				err := b.CreateModel("foo")
				assert.NoError(t, err)

				err = b.CreateModel("bar")
				assert.NoError(t, err)

				err = b.CreateModel("baz")
				assert.NoError(t, err)

				found, err := b.HasModel("bar")
				assert.NoError(t, err)
				assert.True(t, found)

				found, err = b.HasModel("foobar")
				assert.NoError(t, err)
				assert.False(t, found)
			},
		},
		{
			name: "TestDelete",
			test: func(t *testing.T) {
				b := createBackend()
				defer b.Destroy()

				err := b.CreateModel("foo")
				assert.NoError(t, err)

				err = b.DeleteModel("bar")
				{
					concreteErr := &backend.UnknownModelError{}
					assert.ErrorAs(t, err, &concreteErr)
					assert.Equal(t, "bar", concreteErr.ModelID)
				}
				assert.EqualError(t, err, "no model \"bar\" found")

				found, err := b.HasModel("foo")
				assert.NoError(t, err)
				assert.True(t, found)

				err = b.DeleteModel("foo")
				assert.NoError(t, err)

				err = b.DeleteModel("foo")
				{
					concreteErr := &backend.UnknownModelError{}
					assert.ErrorAs(t, err, &concreteErr)
					assert.Equal(t, "foo", concreteErr.ModelID)
				}
				assert.EqualError(t, err, "no model \"foo\" found")
			},
		},
		{
			name: "TestListModels",
			test: func(t *testing.T) {
				b := createBackend()
				defer b.Destroy()

				models, err := b.ListModels(-1, -1)
				assert.NoError(t, err)
				assert.Len(t, models, 0)

				err = b.CreateModel("foo")
				assert.NoError(t, err)

				_, err = b.CreateOrUpdateModelVersion("foo", -1, Data1, false)
				assert.NoError(t, err)

				err = b.CreateModel("bar")
				assert.NoError(t, err)

				for i := 0; i < 15; i++ {
					_, err = b.CreateOrUpdateModelVersion("bar", -1, Data2, true)
					assert.NoError(t, err)
				}

				err = b.CreateModel("baz")
				assert.NoError(t, err)

				models, err = b.ListModels(-1, -1)
				assert.NoError(t, err)
				assert.Len(t, models, 3)

				assert.Equal(t, "bar", models[0])
				assert.Equal(t, "baz", models[1])
				assert.Equal(t, "foo", models[2])

				err = b.DeleteModel("bar")
				assert.NoError(t, err)

				models, err = b.ListModels(-1, -1)
				assert.NoError(t, err)
				assert.Len(t, models, 2)

				assert.Equal(t, "baz", models[0])
				assert.Equal(t, "foo", models[1])
			},
		},
		{
			name: "TestCreateModelVersion",
			test: func(t *testing.T) {
				b := createBackend()
				defer b.Destroy()

				err := b.CreateModel("foo")
				assert.NoError(t, err)

				modelVersion1, err := b.CreateOrUpdateModelVersion("foo", -1, Data1, true)
				assert.NoError(t, err)
				assert.Equal(t, 1, modelVersion1.Number)
				assert.Equal(t, "foo", modelVersion1.ModelID)

				modelVersion2, err := b.CreateOrUpdateModelVersion("foo", -1, Data1, true)
				assert.NoError(t, err)
				assert.Equal(t, 2, modelVersion2.Number)
				assert.Equal(t, "foo", modelVersion1.ModelID)

				for i := 0; i < 20; i++ {
					modelVersionI, err := b.CreateOrUpdateModelVersion("foo", -1, Data2, false)
					assert.NoError(t, err)
					assert.Equal(t, backend.ComputeHash(Data2), modelVersionI.Hash)
				}

				modelVersion23, err := b.CreateOrUpdateModelVersion("foo", -1, Data1, true)
				assert.NoError(t, err)
				assert.Equal(t, 23, modelVersion23.Number)
				assert.Equal(t, backend.ComputeHash(Data1), modelVersion23.Hash)

				modelVersion10, err := b.CreateOrUpdateModelVersion("foo", 10, Data1, true)
				assert.NoError(t, err)
				assert.Equal(t, 10, modelVersion10.Number)
				assert.Equal(t, backend.ComputeHash(Data1), modelVersion10.Hash)
			},
		},
		{
			name: "TestRetrieveModelVersion",
			test: func(t *testing.T) {
				b := createBackend()
				defer b.Destroy()

				err := b.CreateModel("foo")
				assert.NoError(t, err)

				_, err = b.CreateOrUpdateModelVersion("foo", -1, Data1, false)
				assert.NoError(t, err)

				_, err = b.CreateOrUpdateModelVersion("foo", -1, Data2, true)
				assert.NoError(t, err)

				modelVersion1, err := b.RetrieveModelVersionInfo("foo", 1)
				assert.NoError(t, err)
				assert.Equal(t, "foo", modelVersion1.ModelID)
				assert.Equal(t, 1, modelVersion1.Number)
				assert.False(t, modelVersion1.Archive)
				assert.Equal(t, backend.ComputeHash(Data1), modelVersion1.Hash)

				modelVersion1Data, err := b.RetrieveModelVersionData("foo", 2)
				assert.NoError(t, err)
				assert.Equal(t, Data2, modelVersion1Data)

				modelVersion2, err := b.RetrieveModelVersionInfo("foo", 2)
				assert.NoError(t, err)
				assert.Equal(t, "foo", modelVersion2.ModelID)
				assert.Equal(t, 2, modelVersion2.Number)
				assert.True(t, modelVersion2.Archive)
				assert.Equal(t, backend.ComputeHash(Data2), modelVersion2.Hash)

				modelVersion2Data, err := b.RetrieveModelVersionData("foo", 2)
				assert.NoError(t, err)
				assert.Equal(t, Data2, modelVersion2Data)

				_, err = b.RetrieveModelVersionInfo("foo", 3)
				{
					concreteErr := &backend.UnknownModelVersionError{}
					assert.ErrorAs(t, err, &concreteErr)
					assert.Equal(t, "foo", concreteErr.ModelID)
					assert.Equal(t, 3, concreteErr.VersionNumber)
				}
				assert.EqualError(t, err, "no version \"3\" for model \"foo\" found")

				err = b.DeleteModel("foo")
				assert.NoError(t, err)

				found, err := b.HasModel("foo")
				assert.NoError(t, err)
				assert.False(t, found)

				_, err = b.RetrieveModelVersionInfo("foo", 1)
				{
					concreteErr := &backend.UnknownModelVersionError{}
					assert.ErrorAs(t, err, &concreteErr)
					assert.Equal(t, "foo", concreteErr.ModelID)
					assert.Equal(t, 1, concreteErr.VersionNumber)
				}
				assert.EqualError(t, err, "no version \"1\" for model \"foo\" found")

				_, err = b.RetrieveModelVersionData("foo", 2)
				{
					concreteErr := &backend.UnknownModelVersionError{}
					assert.ErrorAs(t, err, &concreteErr)
					assert.Equal(t, "foo", concreteErr.ModelID)
					assert.Equal(t, 2, concreteErr.VersionNumber)
				}
				assert.EqualError(t, err, "no version \"2\" for model \"foo\" found")
			},
		},
		{
			name: "RetrieveModelVersion - Latest",
			test: func(t *testing.T) {
				b := createBackend()
				defer b.Destroy()

				err := b.CreateModel("foo")
				assert.NoError(t, err)

				_, err = b.RetrieveModelVersionInfo("foo", -1)
				{
					concreteErr := &backend.UnknownModelVersionError{}
					assert.ErrorAs(t, err, &concreteErr)
					assert.Equal(t, "foo", concreteErr.ModelID)
				}
				assert.EqualError(t, err, "model \"foo\" doesn't have any version yet")

				_, err = b.RetrieveModelVersionData("foo", -1)
				{
					concreteErr := &backend.UnknownModelVersionError{}
					assert.ErrorAs(t, err, &concreteErr)
					assert.Equal(t, "foo", concreteErr.ModelID)
				}
				assert.EqualError(t, err, "model \"foo\" doesn't have any version yet")

				_, err = b.CreateOrUpdateModelVersion("foo", -1, Data1, true)
				assert.NoError(t, err)

				modelVersion1, err := b.RetrieveModelVersionInfo("foo", -1)
				assert.NoError(t, err)
				assert.Equal(t, "foo", modelVersion1.ModelID)
				assert.Equal(t, 1, modelVersion1.Number)
				assert.True(t, modelVersion1.Archive)
				assert.Equal(t, backend.ComputeHash(Data1), modelVersion1.Hash)

				modelVersion1Data, err := b.RetrieveModelVersionData("foo", -1)
				assert.NoError(t, err)
				assert.Equal(t, Data1, modelVersion1Data)

				_, err = b.CreateOrUpdateModelVersion("foo", -1, Data2, false)
				assert.NoError(t, err)

				modelVersion2, err := b.RetrieveModelVersionInfo("foo", -1)
				assert.NoError(t, err)
				assert.Equal(t, "foo", modelVersion2.ModelID)
				assert.Equal(t, 2, modelVersion2.Number)
				assert.False(t, modelVersion2.Archive)
				assert.Equal(t, backend.ComputeHash(Data2), modelVersion2.Hash)

				modelVersion2Data, err := b.RetrieveModelVersionData("foo", -1)
				assert.NoError(t, err)
				assert.Equal(t, Data2, modelVersion2Data)
			},
		},
		{
			name: "DeleteModelVersion",
			test: func(t *testing.T) {
				b := createBackend()
				defer b.Destroy()

				err := b.CreateModel("foo")
				assert.NoError(t, err)

				_, err = b.CreateOrUpdateModelVersion("foo", -1, Data1, false)
				assert.NoError(t, err)
				versions, err := b.ListModelVersionInfos("foo", -1, -1)
				assert.NoError(t, err)
				assert.Len(t, versions, 1)

				_, err = b.CreateOrUpdateModelVersion("foo", 12, Data2, false)
				assert.NoError(t, err)
				versions, err = b.ListModelVersionInfos("foo", -1, -1)
				assert.NoError(t, err)
				assert.Len(t, versions, 2)

				_, err = b.CreateOrUpdateModelVersion("foo", 3, Data1, false)
				assert.NoError(t, err)
				versions, err = b.ListModelVersionInfos("foo", -1, -1)
				assert.NoError(t, err)
				assert.Len(t, versions, 3)

				_, err = b.CreateOrUpdateModelVersion("foo", -1, Data2, false)
				assert.NoError(t, err)
				versions, err = b.ListModelVersionInfos("foo", -1, -1)
				assert.NoError(t, err)
				assert.Len(t, versions, 4)

				assert.Equal(t, "foo", versions[0].ModelID)
				assert.Equal(t, 1, versions[0].Number)
				assert.Equal(t, backend.ComputeHash(Data1), versions[0].Hash)

				assert.Equal(t, "foo", versions[1].ModelID)
				assert.Equal(t, 3, versions[1].Number)
				assert.Equal(t, backend.ComputeHash(Data1), versions[1].Hash)

				assert.Equal(t, "foo", versions[2].ModelID)
				assert.Equal(t, 12, versions[2].Number)
				assert.Equal(t, backend.ComputeHash(Data2), versions[2].Hash)

				assert.Equal(t, "foo", versions[3].ModelID)
				assert.Equal(t, 13, versions[3].Number)
				assert.Equal(t, backend.ComputeHash(Data2), versions[3].Hash)

				err = b.DeleteModelVersion("foo", 3)
				assert.NoError(t, err)

				versions, err = b.ListModelVersionInfos("foo", -1, -1)
				assert.NoError(t, err)
				assert.Len(t, versions, 3)

				err = b.DeleteModelVersion("foo", 13)
				assert.NoError(t, err)

				versions, err = b.ListModelVersionInfos("foo", -1, -1)
				assert.NoError(t, err)
				assert.Len(t, versions, 2)

				modelLatestVersionInfo, err := b.RetrieveModelVersionInfo("foo", -1)
				assert.NoError(t, err)
				assert.Equal(t, "foo", modelLatestVersionInfo.ModelID)
				assert.Equal(t, 12, modelLatestVersionInfo.Number)
				assert.Equal(t, backend.ComputeHash(Data2), modelLatestVersionInfo.Hash)
			},
		},
		{
			name: "TestListModelVersions",
			test: func(t *testing.T) {
				b := createBackend()
				defer b.Destroy()

				err := b.CreateModel("foo")
				assert.NoError(t, err)

				for i := 0; i < 20; i++ {
					_, err := b.CreateOrUpdateModelVersion("foo", -1, Data1, false)
					assert.NoError(t, err)
				}

				versions, err := b.ListModelVersionInfos("foo", -1, -1)
				assert.NoError(t, err)
				assert.Len(t, versions, 20)

				for i, version := range versions {
					assert.Equal(t, "foo", version.ModelID)
					assert.Equal(t, i+1, version.Number)
					assert.NotNil(t, version.Hash)
				}

				versions, err = b.ListModelVersionInfos("foo", 0, 2)
				assert.NoError(t, err)
				assert.Len(t, versions, 2)
				assert.Equal(t, 1, versions[0].Number)
				assert.Equal(t, 2, versions[1].Number)

				versions, err = b.ListModelVersionInfos("foo", 2, 3)
				assert.NoError(t, err)
				assert.Len(t, versions, 3)
				assert.Equal(t, 3, versions[0].Number)
				assert.Equal(t, 4, versions[1].Number)
				assert.Equal(t, 5, versions[2].Number)
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, c.test)
	}
}
