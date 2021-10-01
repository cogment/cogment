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

package hybrid

import (
	"testing"

	"github.com/cogment/cogment-model-registry/backend"
	"github.com/cogment/cogment-model-registry/backend/db"
	"github.com/cogment/cogment-model-registry/backend/fs"
	"github.com/cogment/cogment-model-registry/backend/test"
	"github.com/stretchr/testify/assert"
)

func TestSuiteHybridBackend(t *testing.T) {
	test.RunSuite(t, func() backend.Backend {
		fsBackend, err := fs.CreateBackend(t.TempDir())
		assert.NoError(t, err)
		dbBackend, err := db.CreateBackend()
		assert.NoError(t, err)
		b, err := CreateBackend(dbBackend, fsBackend)
		assert.NoError(t, err)
		return b
	}, func(b backend.Backend) {
		hb := b.(*hybridBackend)
		hb.archive.Destroy()
		hb.transient.Destroy()
		hb.Destroy()
	})
}

func TestInitialSync(t *testing.T) {

	versionUserData := make(map[string]string)
	versionUserData["version_test1"] = "version_test1"
	versionUserData["version_test2"] = "version_test2"
	versionUserData["version_test3"] = "version_test3"

	modelUserData := make(map[string]string)
	modelUserData["model_test1"] = "model_test1"
	modelUserData["model_test2"] = "model_test2"
	modelUserData["model_test3"] = "model_test3"

	archiveB, err := db.CreateBackend()
	assert.NoError(t, err)
	{
		transientB, err := db.CreateBackend()
		assert.NoError(t, err)
		b, err := CreateBackend(transientB, archiveB)
		assert.NoError(t, err)

		_, err = b.CreateOrUpdateModel(backend.ModelInfo{
			ModelID:  "model1",
			UserData: modelUserData,
		})
		assert.NoError(t, err)
		_, err = b.CreateOrUpdateModelVersion("model1", backend.VersionArgs{
			VersionNumber: -1,
			Data:          test.Data1,
			UserData:      versionUserData,
			Archived:      true,
		})
		assert.NoError(t, err)
		_, err = b.CreateOrUpdateModelVersion("model1", backend.VersionArgs{
			VersionNumber: -1,
			Data:          test.Data2,
			UserData:      versionUserData,
			Archived:      false,
		})
		assert.NoError(t, err)
		_, err = b.CreateOrUpdateModelVersion("model1", backend.VersionArgs{
			VersionNumber: -1,
			Data:          test.Data2,
			UserData:      versionUserData,
			Archived:      false,
		})
		assert.NoError(t, err)
		_, err = b.CreateOrUpdateModelVersion("model1", backend.VersionArgs{
			VersionNumber: -1,
			Data:          test.Data1,
			UserData:      versionUserData,
			Archived:      true,
		})
		assert.NoError(t, err)

		_, err = b.CreateOrUpdateModel(backend.ModelInfo{
			ModelID:  "model2",
			UserData: modelUserData,
		})
		assert.NoError(t, err)
		_, err = b.CreateOrUpdateModelVersion("model2", backend.VersionArgs{
			VersionNumber: -1,
			Data:          test.Data2,
			UserData:      versionUserData,
			Archived:      false,
		})
		assert.NoError(t, err)
		_, err = b.CreateOrUpdateModelVersion("model2", backend.VersionArgs{
			VersionNumber: -1,
			Data:          test.Data2,
			UserData:      versionUserData,
			Archived:      false,
		})
		assert.NoError(t, err)
		_, err = b.CreateOrUpdateModelVersion("model2", backend.VersionArgs{
			VersionNumber: -1,
			Data:          test.Data2,
			UserData:      versionUserData,
			Archived:      false,
		})
		assert.NoError(t, err)
		_, err = b.CreateOrUpdateModelVersion("model2", backend.VersionArgs{
			VersionNumber: -1,
			Data:          test.Data2,
			UserData:      versionUserData,
			Archived:      false,
		})
		assert.NoError(t, err)

		models, err := b.ListModels(-1, -1)
		assert.NoError(t, err)
		assert.Len(t, models, 2)

		model1Versions, err := b.ListModelVersionInfos("model1", -1, -1)
		assert.NoError(t, err)
		assert.Len(t, model1Versions, 4)
		assert.Equal(t, model1Versions[0].VersionNumber, 1)
		assert.Equal(t, model1Versions[1].VersionNumber, 2)
		assert.Equal(t, model1Versions[2].VersionNumber, 3)
		assert.Equal(t, model1Versions[3].VersionNumber, 4)

		model2Versions, err := b.ListModelVersionInfos("model2", -1, -1)
		assert.NoError(t, err)
		assert.Len(t, model2Versions, 4)
	}
	{
		transientB, err := db.CreateBackend()
		assert.NoError(t, err)
		b, err := CreateBackend(transientB, archiveB)
		assert.NoError(t, err)

		models, err := b.ListModels(-1, -1)
		assert.NoError(t, err)
		assert.Len(t, models, 2)
		assert.Equal(t, models[0], "model1")
		assert.Equal(t, models[1], "model2")

		model1Versions, err := b.ListModelVersionInfos("model1", -1, -1)
		assert.NoError(t, err)
		assert.Len(t, model1Versions, 2)
		assert.Equal(t, model1Versions[0].VersionNumber, 1)
		assert.Equal(t, model1Versions[1].VersionNumber, 4)

		model2Versions, err := b.ListModelVersionInfos("model2", -1, -1)
		assert.NoError(t, err)
		assert.Len(t, model2Versions, 0)
	}
}
