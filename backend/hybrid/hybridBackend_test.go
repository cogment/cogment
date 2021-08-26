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

	"github.com/cogment/model-registry/backend"
	"github.com/cogment/model-registry/backend/db"
	"github.com/cogment/model-registry/backend/fs"
	"github.com/cogment/model-registry/backend/test"
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
	})
}

func TestInitialSync(t *testing.T) {
	archiveB, err := db.CreateBackend()
	assert.NoError(t, err)
	{
		transientB, err := db.CreateBackend()
		assert.NoError(t, err)
		b, err := CreateBackend(transientB, archiveB)
		assert.NoError(t, err)

		err = b.CreateModel("model1")
		assert.NoError(t, err)
		_, err = b.CreateOrUpdateModelVersion("model1", -1, test.Data1, true)
		assert.NoError(t, err)
		_, err = b.CreateOrUpdateModelVersion("model1", -1, test.Data2, false)
		assert.NoError(t, err)
		_, err = b.CreateOrUpdateModelVersion("model1", -1, test.Data2, false)
		assert.NoError(t, err)
		_, err = b.CreateOrUpdateModelVersion("model1", -1, test.Data1, true)
		assert.NoError(t, err)

		err = b.CreateModel("model2")
		assert.NoError(t, err)
		_, err = b.CreateOrUpdateModelVersion("model2", -1, test.Data2, false)
		assert.NoError(t, err)
		_, err = b.CreateOrUpdateModelVersion("model2", -1, test.Data2, false)
		assert.NoError(t, err)
		_, err = b.CreateOrUpdateModelVersion("model2", -1, test.Data2, false)
		assert.NoError(t, err)
		_, err = b.CreateOrUpdateModelVersion("model2", -1, test.Data2, false)
		assert.NoError(t, err)

		models, err := b.ListModels(-1, -1)
		assert.NoError(t, err)
		assert.Len(t, models, 2)

		model1Versions, err := b.ListModelVersionInfos("model1", -1, -1)
		assert.NoError(t, err)
		assert.Len(t, model1Versions, 4)
		assert.Equal(t, model1Versions[0].Number, 1)
		assert.Equal(t, model1Versions[1].Number, 2)
		assert.Equal(t, model1Versions[2].Number, 3)
		assert.Equal(t, model1Versions[3].Number, 4)

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
		assert.Equal(t, model1Versions[0].Number, 1)
		assert.Equal(t, model1Versions[1].Number, 4)

		model2Versions, err := b.ListModelVersionInfos("model2", -1, -1)
		assert.NoError(t, err)
		assert.Len(t, model2Versions, 0)
	}
}
