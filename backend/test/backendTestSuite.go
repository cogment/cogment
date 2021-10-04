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
	"sync"
	"testing"
	"time"

	"github.com/cogment/cogment-model-registry/backend"
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
	versionUserData := make(map[string]string)
	versionUserData["version_test1"] = "version_test1"
	versionUserData["version_test2"] = "version_test2"
	versionUserData["version_test3"] = "version_test3"

	modelUserData := make(map[string]string)
	modelUserData["model_test1"] = "model_test1"
	modelUserData["model_test2"] = "model_test2"
	modelUserData["model_test3"] = "model_test3"

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

				_, err := b.CreateOrUpdateModel(backend.ModelInfo{
					ModelID:  "foo",
					UserData: modelUserData,
				})
				assert.NoError(t, err)

				// Create a another one should succeed
				_, err = b.CreateOrUpdateModel(backend.ModelInfo{
					ModelID:  "bar",
					UserData: modelUserData,
				})
				assert.NoError(t, err)
			},
		},
		{
			name: "TestHasModel",
			test: func(t *testing.T) {
				b := createBackend()
				defer b.Destroy()

				_, err := b.CreateOrUpdateModel(backend.ModelInfo{
					ModelID:  "foo",
					UserData: modelUserData,
				})
				assert.NoError(t, err)

				_, err = b.CreateOrUpdateModel(backend.ModelInfo{
					ModelID:  "bar",
					UserData: modelUserData,
				})
				assert.NoError(t, err)

				_, err = b.CreateOrUpdateModel(backend.ModelInfo{
					ModelID:  "baz",
					UserData: modelUserData,
				})
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

				{
					_, err := b.CreateOrUpdateModel(backend.ModelInfo{
						ModelID:  "foo",
						UserData: modelUserData,
					})
					assert.NoError(t, err)
				}
				{
					err := b.DeleteModel("bar")
					concreteErr := &backend.UnknownModelError{}
					assert.ErrorAs(t, err, &concreteErr)
					assert.Equal(t, "bar", concreteErr.ModelID)
					assert.EqualError(t, err, `no model "bar" found`)
				}
				{
					found, err := b.HasModel("foo")
					assert.NoError(t, err)
					assert.True(t, found)
				}
				{
					err := b.DeleteModel("foo")
					assert.NoError(t, err)
				}
				{
					err := b.DeleteModel("foo")
					concreteErr := &backend.UnknownModelError{}
					assert.ErrorAs(t, err, &concreteErr)
					assert.Equal(t, "foo", concreteErr.ModelID)
					assert.EqualError(t, err, `no model "foo" found`)
				}
				{
					found, err := b.HasModel("foo")
					assert.NoError(t, err)
					assert.False(t, found)
				}
				{
					_, err := b.CreateOrUpdateModel(backend.ModelInfo{
						ModelID:  "foo",
						UserData: modelUserData,
					})
					assert.NoError(t, err)
				}
			},
		},
		{
			name: "TestListModels",
			test: func(t *testing.T) {
				b := createBackend()
				defer b.Destroy()

				models, err := b.ListModels(0, 0)
				assert.NoError(t, err)
				assert.Len(t, models, 0)

				_, err = b.CreateOrUpdateModel(backend.ModelInfo{
					ModelID:  "foo",
					UserData: modelUserData,
				})
				assert.NoError(t, err)

				_, err = b.CreateOrUpdateModelVersion("foo", backend.VersionArgs{
					VersionNumber:     -1,
					CreationTimestamp: time.Now(),
					Data:              Data1,
					DataHash:          backend.ComputeSHA256Hash(Data1),
					Archived:          false,
					UserData:          versionUserData,
				})
				assert.NoError(t, err)

				_, err = b.CreateOrUpdateModel(backend.ModelInfo{
					ModelID:  "bar",
					UserData: modelUserData,
				})
				assert.NoError(t, err)

				for i := 0; i < 15; i++ {
					_, err = b.CreateOrUpdateModelVersion("bar", backend.VersionArgs{
						VersionNumber:     -1,
						CreationTimestamp: time.Now(),
						Data:              Data2,
						DataHash:          backend.ComputeSHA256Hash(Data2),
						Archived:          true,
						UserData:          versionUserData,
					})
					assert.NoError(t, err)
				}

				_, err = b.CreateOrUpdateModel(backend.ModelInfo{
					ModelID:  "baz",
					UserData: modelUserData,
				})
				assert.NoError(t, err)

				models, err = b.ListModels(0, 0)
				assert.NoError(t, err)
				assert.Len(t, models, 3)

				assert.Equal(t, "bar", models[0])
				assert.Equal(t, "baz", models[1])
				assert.Equal(t, "foo", models[2])

				err = b.DeleteModel("bar")
				assert.NoError(t, err)

				models, err = b.ListModels(0, 0)
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

				_, err := b.CreateOrUpdateModel(backend.ModelInfo{
					ModelID:  "foo",
					UserData: modelUserData,
				})
				assert.NoError(t, err)

				modelVersion1, err := b.CreateOrUpdateModelVersion("foo", backend.VersionArgs{
					VersionNumber:     -1,
					CreationTimestamp: time.Now(),
					Data:              Data1,
					DataHash:          backend.ComputeSHA256Hash(Data1),
					Archived:          true,
					UserData:          versionUserData,
				})
				assert.NoError(t, err)
				assert.Equal(t, 1, modelVersion1.VersionNumber)
				assert.Equal(t, "foo", modelVersion1.ModelID)

				modelVersion2, err := b.CreateOrUpdateModelVersion("foo", backend.VersionArgs{
					VersionNumber:     -1,
					CreationTimestamp: time.Now(),
					Data:              Data1,
					DataHash:          backend.ComputeSHA256Hash(Data1),
					Archived:          true,
					UserData:          versionUserData,
				})
				assert.NoError(t, err)
				assert.Equal(t, 2, modelVersion2.VersionNumber)
				assert.Equal(t, "foo", modelVersion1.ModelID)

				for i := 0; i < 20; i++ {
					modelVersionI, err := b.CreateOrUpdateModelVersion("foo", backend.VersionArgs{
						VersionNumber:     -1,
						CreationTimestamp: time.Now(),
						Data:              Data2,
						DataHash:          backend.ComputeSHA256Hash(Data2),
						Archived:          false,
						UserData:          versionUserData,
					})
					assert.NoError(t, err)
					assert.Equal(t, backend.ComputeSHA256Hash(Data2), modelVersionI.DataHash)
					assert.Equal(t, len(Data2), modelVersionI.DataSize)
				}

				modelVersion23, err := b.CreateOrUpdateModelVersion("foo", backend.VersionArgs{
					VersionNumber:     -1,
					CreationTimestamp: time.Now(),
					Data:              Data1,
					DataHash:          backend.ComputeSHA256Hash(Data1),
					Archived:          true,
					UserData:          versionUserData,
				})
				assert.NoError(t, err)
				assert.Equal(t, 23, modelVersion23.VersionNumber)
				assert.Equal(t, backend.ComputeSHA256Hash(Data1), modelVersion23.DataHash)
				assert.Equal(t, len(Data1), modelVersion23.DataSize)

				modelVersion10, err := b.CreateOrUpdateModelVersion("foo", backend.VersionArgs{
					VersionNumber:     10,
					CreationTimestamp: time.Now(),
					Data:              Data1,
					DataHash:          backend.ComputeSHA256Hash(Data1),
					Archived:          true,
					UserData:          versionUserData,
				})
				assert.NoError(t, err)
				assert.Equal(t, 10, modelVersion10.VersionNumber)
				assert.Equal(t, backend.ComputeSHA256Hash(Data1), modelVersion10.DataHash)
				assert.Equal(t, len(Data1), modelVersion10.DataSize)
			},
		},
		{
			name: "TestRetrieveModelVersion",
			test: func(t *testing.T) {
				b := createBackend()
				defer b.Destroy()

				_, err := b.CreateOrUpdateModel(backend.ModelInfo{
					ModelID:  "foo",
					UserData: modelUserData,
				})
				assert.NoError(t, err)

				_, err = b.CreateOrUpdateModelVersion("foo", backend.VersionArgs{
					VersionNumber:     -1,
					CreationTimestamp: time.Now(),
					Data:              Data1,
					DataHash:          backend.ComputeSHA256Hash(Data1),
					Archived:          false,
					UserData:          versionUserData,
				})
				assert.NoError(t, err)

				_, err = b.CreateOrUpdateModelVersion("foo", backend.VersionArgs{
					VersionNumber:     -1,
					CreationTimestamp: time.Now(),
					Data:              Data2,
					DataHash:          backend.ComputeSHA256Hash(Data2),
					Archived:          true,
					UserData:          versionUserData,
				})
				assert.NoError(t, err)

				modelVersion1, err := b.RetrieveModelVersionInfo("foo", 1)
				assert.NoError(t, err)
				assert.Equal(t, "foo", modelVersion1.ModelID)
				assert.Equal(t, 1, modelVersion1.VersionNumber)
				assert.False(t, modelVersion1.Archived)
				assert.Equal(t, backend.ComputeSHA256Hash(Data1), modelVersion1.DataHash)
				assert.Equal(t, len(Data1), modelVersion1.DataSize)

				modelVersion1Data, err := b.RetrieveModelVersionData("foo", 2)
				assert.NoError(t, err)
				assert.Equal(t, Data2, modelVersion1Data)

				modelVersion2, err := b.RetrieveModelVersionInfo("foo", 2)
				assert.NoError(t, err)
				assert.Equal(t, "foo", modelVersion2.ModelID)
				assert.Equal(t, 2, modelVersion2.VersionNumber)
				assert.True(t, modelVersion2.Archived)
				assert.Equal(t, backend.ComputeSHA256Hash(Data2), modelVersion2.DataHash)
				assert.Equal(t, len(Data2), modelVersion2.DataSize)

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
				assert.EqualError(t, err, `no version "3" for model "foo" found`)

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
				assert.EqualError(t, err, `no version "1" for model "foo" found`)

				_, err = b.RetrieveModelVersionData("foo", 2)
				{
					concreteErr := &backend.UnknownModelVersionError{}
					assert.ErrorAs(t, err, &concreteErr)
					assert.Equal(t, "foo", concreteErr.ModelID)
					assert.Equal(t, 2, concreteErr.VersionNumber)
				}
				assert.EqualError(t, err, `no version "2" for model "foo" found`)
			},
		},
		{
			name: "RetrieveModelVersion - Latest",
			test: func(t *testing.T) {
				b := createBackend()
				defer b.Destroy()

				_, err := b.CreateOrUpdateModel(backend.ModelInfo{
					ModelID:  "foo",
					UserData: modelUserData,
				})
				assert.NoError(t, err)

				_, err = b.RetrieveModelVersionInfo("foo", -1)
				{
					concreteErr := &backend.UnknownModelVersionError{}
					assert.ErrorAs(t, err, &concreteErr)
					assert.Equal(t, "foo", concreteErr.ModelID)
				}
				assert.EqualError(t, err, `model "foo" doesn't have any version yet`)

				_, err = b.RetrieveModelVersionData("foo", -1)
				{
					concreteErr := &backend.UnknownModelVersionError{}
					assert.ErrorAs(t, err, &concreteErr)
					assert.Equal(t, "foo", concreteErr.ModelID)
				}
				assert.EqualError(t, err, `model "foo" doesn't have any version yet`)

				_, err = b.CreateOrUpdateModelVersion("foo", backend.VersionArgs{
					VersionNumber:     -1,
					CreationTimestamp: time.Now(),
					Data:              Data1,
					DataHash:          backend.ComputeSHA256Hash(Data1),
					Archived:          true,
					UserData:          versionUserData,
				})
				assert.NoError(t, err)

				modelVersion1, err := b.RetrieveModelVersionInfo("foo", -1)
				assert.NoError(t, err)
				assert.Equal(t, "foo", modelVersion1.ModelID)
				assert.Equal(t, 1, modelVersion1.VersionNumber)
				assert.True(t, modelVersion1.Archived)
				assert.Equal(t, backend.ComputeSHA256Hash(Data1), modelVersion1.DataHash)
				assert.Equal(t, len(Data1), modelVersion1.DataSize)

				modelVersion1Data, err := b.RetrieveModelVersionData("foo", -1)
				assert.NoError(t, err)
				assert.Equal(t, Data1, modelVersion1Data)

				_, err = b.CreateOrUpdateModelVersion("foo", backend.VersionArgs{
					VersionNumber:     -1,
					CreationTimestamp: time.Now(),
					Data:              Data2,
					DataHash:          backend.ComputeSHA256Hash(Data2),
					Archived:          false,
					UserData:          versionUserData,
				})
				assert.NoError(t, err)

				modelVersion2, err := b.RetrieveModelVersionInfo("foo", -1)
				assert.NoError(t, err)
				assert.Equal(t, "foo", modelVersion2.ModelID)
				assert.Equal(t, 2, modelVersion2.VersionNumber)
				assert.False(t, modelVersion2.Archived)
				assert.Equal(t, backend.ComputeSHA256Hash(Data2), modelVersion2.DataHash)
				assert.Equal(t, len(Data2), modelVersion2.DataSize)

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

				_, err := b.CreateOrUpdateModel(backend.ModelInfo{
					ModelID:  "foo",
					UserData: modelUserData,
				})
				assert.NoError(t, err)

				_, err = b.CreateOrUpdateModelVersion("foo", backend.VersionArgs{
					VersionNumber:     -1,
					CreationTimestamp: time.Now(),
					Data:              Data1,
					DataHash:          backend.ComputeSHA256Hash(Data1),
					Archived:          false,
					UserData:          versionUserData,
				})
				assert.NoError(t, err)
				versions, err := b.ListModelVersionInfos("foo", 0, 0)
				assert.NoError(t, err)
				assert.Len(t, versions, 1)

				_, err = b.CreateOrUpdateModelVersion("foo", backend.VersionArgs{
					VersionNumber:     12,
					CreationTimestamp: time.Now(),
					Data:              Data2,
					DataHash:          backend.ComputeSHA256Hash(Data2),
					Archived:          false,
					UserData:          versionUserData,
				})
				assert.NoError(t, err)
				versions, err = b.ListModelVersionInfos("foo", 0, 0)
				assert.NoError(t, err)
				assert.Len(t, versions, 2)

				_, err = b.CreateOrUpdateModelVersion("foo", backend.VersionArgs{
					VersionNumber:     3,
					CreationTimestamp: time.Now(),
					Data:              Data1,
					DataHash:          backend.ComputeSHA256Hash(Data1),
					Archived:          false,
					UserData:          versionUserData,
				})
				assert.NoError(t, err)
				versions, err = b.ListModelVersionInfos("foo", 0, 0)
				assert.NoError(t, err)
				assert.Len(t, versions, 3)

				_, err = b.CreateOrUpdateModelVersion("foo", backend.VersionArgs{
					VersionNumber:     -1,
					CreationTimestamp: time.Now(),
					Data:              Data2,
					DataHash:          backend.ComputeSHA256Hash(Data2),
					Archived:          false,
					UserData:          versionUserData,
				})
				assert.NoError(t, err)
				versions, err = b.ListModelVersionInfos("foo", 0, 0)
				assert.NoError(t, err)
				assert.Len(t, versions, 4)

				assert.Equal(t, "foo", versions[0].ModelID)
				assert.Equal(t, 1, versions[0].VersionNumber)
				assert.Equal(t, backend.ComputeSHA256Hash(Data1), versions[0].DataHash)
				assert.Equal(t, len(Data1), versions[0].DataSize)

				assert.Equal(t, "foo", versions[1].ModelID)
				assert.Equal(t, 3, versions[1].VersionNumber)
				assert.Equal(t, backend.ComputeSHA256Hash(Data1), versions[1].DataHash)
				assert.Equal(t, len(Data1), versions[1].DataSize)

				assert.Equal(t, "foo", versions[2].ModelID)
				assert.Equal(t, 12, versions[2].VersionNumber)
				assert.Equal(t, backend.ComputeSHA256Hash(Data2), versions[2].DataHash)
				assert.Equal(t, len(Data2), versions[2].DataSize)

				assert.Equal(t, "foo", versions[3].ModelID)
				assert.Equal(t, 13, versions[3].VersionNumber)
				assert.Equal(t, backend.ComputeSHA256Hash(Data2), versions[3].DataHash)
				assert.Equal(t, len(Data2), versions[3].DataSize)

				err = b.DeleteModelVersion("foo", 3)
				assert.NoError(t, err)

				versions, err = b.ListModelVersionInfos("foo", 0, 0)
				assert.NoError(t, err)
				assert.Len(t, versions, 3)

				err = b.DeleteModelVersion("foo", 13)
				assert.NoError(t, err)

				versions, err = b.ListModelVersionInfos("foo", 0, 0)
				assert.NoError(t, err)
				assert.Len(t, versions, 2)

				modelLatestVersionInfo, err := b.RetrieveModelVersionInfo("foo", -1)
				assert.NoError(t, err)
				assert.Equal(t, "foo", modelLatestVersionInfo.ModelID)
				assert.Equal(t, 12, modelLatestVersionInfo.VersionNumber)
				assert.Equal(t, backend.ComputeSHA256Hash(Data2), modelLatestVersionInfo.DataHash)
				assert.Equal(t, len(Data2), modelLatestVersionInfo.DataSize)
			},
		},
		{
			name: "TestListModelVersions",
			test: func(t *testing.T) {
				b := createBackend()
				defer b.Destroy()

				_, err := b.CreateOrUpdateModel(backend.ModelInfo{
					ModelID:  "foo",
					UserData: modelUserData,
				})
				assert.NoError(t, err)

				for i := 0; i < 20; i++ {
					_, err := b.CreateOrUpdateModelVersion("foo", backend.VersionArgs{
						VersionNumber:     -1,
						CreationTimestamp: time.Now(),
						Data:              Data1,
						Archived:          false,
						UserData:          versionUserData,
					})
					assert.NoError(t, err)
				}

				versions, err := b.ListModelVersionInfos("foo", 0, 0)
				assert.NoError(t, err)
				assert.Len(t, versions, 20)

				for i, version := range versions {
					assert.Equal(t, "foo", version.ModelID)
					assert.Equal(t, i+1, version.VersionNumber)
					assert.NotNil(t, version.DataHash)
				}

				versions, err = b.ListModelVersionInfos("foo", 0, 2)
				assert.NoError(t, err)
				assert.Len(t, versions, 2)
				assert.Equal(t, 1, versions[0].VersionNumber)
				assert.Equal(t, 2, versions[1].VersionNumber)

				versions, err = b.ListModelVersionInfos("foo", 2, 3)
				assert.NoError(t, err)
				assert.Len(t, versions, 3)
				assert.Equal(t, 3, versions[0].VersionNumber)
				assert.Equal(t, 4, versions[1].VersionNumber)
				assert.Equal(t, 5, versions[2].VersionNumber)
			},
		},
		{
			name: "TestConcurrentCreateAndRetrieveModelVersions",
			test: func(t *testing.T) {
				b := createBackend()
				defer b.Destroy()

				_, err := b.CreateOrUpdateModel(backend.ModelInfo{ModelID: "foo"})
				assert.NoError(t, err)

				_, err = b.CreateOrUpdateModel(backend.ModelInfo{ModelID: "bar"})
				assert.NoError(t, err)

				wg := new(sync.WaitGroup)
				oneFooCreationDone := make(chan struct{})
				oneBarCreationDone := make(chan struct{})

				// 5 "foo" creations in parallel
				for i := 0; i < 5; i++ {
					wg.Add(1)
					i := i
					go func() {
						defer wg.Done()
						_, err = b.CreateOrUpdateModelVersion("foo", backend.VersionArgs{
							VersionNumber:     -1,
							CreationTimestamp: time.Now(),
							Data:              Data1,
							DataHash:          backend.ComputeSHA256Hash(Data1),
							Archived:          i%2 == 0,
						})
						assert.NoError(t, err)

						go func() { oneFooCreationDone <- struct{}{} }()
					}()
				}

				// 5 "bar" creations in parallel
				for i := 0; i < 5; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						_, err = b.CreateOrUpdateModelVersion("bar", backend.VersionArgs{
							VersionNumber:     -1,
							CreationTimestamp: time.Now(),
							Data:              Data2,
							DataHash:          backend.ComputeSHA256Hash(Data2),
							Archived:          i%2 == 0,
						})
						assert.NoError(t, err)

						go func() { oneBarCreationDone <- struct{}{} }()
					}()
				}

				// 5 "foo" retrievals in parallel
				<-oneFooCreationDone
				for i := 0; i < 5; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						fooVersion, err := b.RetrieveModelVersionInfo("foo", -1)
						assert.NoError(t, err)
						assert.Equal(t, "foo", fooVersion.ModelID)
						assert.Equal(t, backend.ComputeSHA256Hash(Data1), fooVersion.DataHash)
						assert.Equal(t, len(Data1), fooVersion.DataSize)
					}()
				}

				// 5 "bar" retrievals in parallel
				<-oneBarCreationDone
				for i := 0; i < 5; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						barVersion, err := b.RetrieveModelVersionInfo("bar", -1)
						assert.NoError(t, err)
						assert.Equal(t, "bar", barVersion.ModelID)
						assert.Equal(t, backend.ComputeSHA256Hash(Data2), barVersion.DataHash)
						assert.Equal(t, len(Data2), barVersion.DataSize)
					}()
				}

				wg.Wait()
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, c.test)
	}
}
