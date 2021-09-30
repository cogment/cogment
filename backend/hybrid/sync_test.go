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
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/cogment/cogment-model-registry/backend"
	"github.com/cogment/cogment-model-registry/backend/db"
	"github.com/cogment/cogment-model-registry/backend/fs"
	"github.com/stretchr/testify/assert"
)

func generateRandomBytes(size uint) (blk []byte, err error) {
	blk = make([]byte, size)
	_, err = rand.Read(blk)
	return blk, err
}

func populateDirWithModels(dir string, modelsCount uint, versionsPerModelCount uint, dataPerVersionSize uint) error {
	b, err := fs.CreateBackend(dir)
	defer b.Destroy()
	if err != nil {
		return err
	}
	for modelIdx := uint(0); modelIdx < modelsCount; modelIdx++ {
		modelID := fmt.Sprintf("model%d", modelIdx)
		_, err = b.CreateOrUpdateModel(backend.ModelInfo{ModelID: modelID})
		if err != nil {
			return err
		}
		for versionIdx := uint(0); versionIdx < versionsPerModelCount; versionIdx++ {
			data, err := generateRandomBytes(dataPerVersionSize)
			if err != nil {
				return err
			}
			_, err = b.CreateOrUpdateModelVersion(modelID, backend.VersionInfoArgs{
				VersionNumber: -1,
				Data:          data,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func BenchmarkSyncFsToDb(b *testing.B) {
	b.ReportAllocs()
	rootDir := b.TempDir()
	err := populateDirWithModels(rootDir, 10, 20, 1024*1024)
	assert.NoError(b, err)

	for i := 0; i < b.N; i++ {
		dbB, err := db.CreateBackend()
		defer dbB.Destroy()
		assert.NoError(b, err)

		fsB, err := fs.CreateBackend(rootDir)
		defer fsB.Destroy()
		assert.NoError(b, err)

		err = Sync(dbB, fsB)
		assert.NoError(b, err)
	}
}
