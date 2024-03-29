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

package memorycache

import (
	"testing"
	"time"

	"github.com/cogment/cogment/services/registry/backend"
	"github.com/cogment/cogment/services/registry/backend/files"
	"github.com/cogment/cogment/services/registry/backend/test"
	"github.com/stretchr/testify/assert"
)

func TestSuiteMemoryCacheOverFsBackend(t *testing.T) {
	test.RunSuite(t, func() backend.Backend {
		fsBackend, err := files.CreateBackend(t.TempDir())
		assert.NoError(t, err)

		b, err := CreateBackend(DefaultVersionCacheConfiguration, fsBackend)
		assert.NoError(t, err)
		return b
	}, func(b backend.Backend) {
		mcb := b.(*memoryCacheBackend)
		mcb.archive.Destroy()
		mcb.Destroy()
	})
}

func BenchmarkMemoryCacheOverFsBackend(b *testing.B) {
	test.RunBenchmarkSuite(b, func() backend.Backend {
		fsBackend, err := files.CreateBackend(b.TempDir())
		assert.NoError(b, err)

		bck, err := CreateBackend(DefaultVersionCacheConfiguration, fsBackend)
		assert.NoError(b, err)
		return bck
	}, func(bck backend.Backend) {
		mcb := bck.(*memoryCacheBackend)
		mcb.archive.Destroy()
		mcb.Destroy()
	})
}

func TestSmallMaxItems(t *testing.T) {
	fsBackend, err := files.CreateBackend(t.TempDir())
	assert.NoError(t, err)
	defer fsBackend.Destroy()

	b, err := CreateBackend(VersionCacheConfiguration{MaxItems: 2}, fsBackend)
	assert.NoError(t, err)
	defer b.Destroy()

	_, err = b.CreateOrUpdateModel(backend.ModelInfo{ModelID: "foo"})
	assert.NoError(t, err)

	assert.Len(t, test.Data1, 751)
	data1Hash := backend.ComputeSHA256Hash(test.Data1)

	_, err = b.CreateOrUpdateModelVersion(
		"foo",
		backend.VersionArgs{Archived: false, DataHash: data1Hash, Data: test.Data1},
	)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	_, err = b.CreateOrUpdateModelVersion(
		"foo",
		backend.VersionArgs{Archived: false, DataHash: data1Hash, Data: test.Data1},
	)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	versionInfo, err := b.RetrieveModelVersionInfo("foo", 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, int(versionInfo.VersionNumber))
	assert.Equal(t, data1Hash, versionInfo.DataHash)

	_, err = b.CreateOrUpdateModelVersion(
		"foo",
		backend.VersionArgs{Archived: false, DataHash: data1Hash, Data: test.Data1},
	)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	versionInfo, err = b.RetrieveModelVersionInfo("foo", 3)
	assert.NoError(t, err)
	assert.Equal(t, 3, int(versionInfo.VersionNumber))
	assert.Equal(t, data1Hash, versionInfo.DataHash)

	versionInfos, err := b.ListModelVersionInfos("foo", 0, -1)
	assert.NoError(t, err)
	assert.Len(t, versionInfos, 2)
	assert.Equal(t, 2, int(versionInfos[0].VersionNumber))
	assert.Equal(t, 3, int(versionInfos[1].VersionNumber))
}

func TestModelAddedInArchiveBackend(t *testing.T) {
	fsBackend, err := files.CreateBackend(t.TempDir())
	assert.NoError(t, err)
	defer fsBackend.Destroy()

	b, err := CreateBackend(VersionCacheConfiguration{MaxItems: 2000}, fsBackend)
	assert.NoError(t, err)
	defer b.Destroy()

	_, err = b.CreateOrUpdateModel(backend.ModelInfo{ModelID: "foo"})
	assert.NoError(t, err)

	_, err = b.RetrieveModelVersionInfo("foo", -1)
	concreteErr := &backend.UnknownModelVersionError{}
	assert.ErrorAs(t, err, &concreteErr)
	assert.Equal(t, "foo", concreteErr.ModelID)
	assert.Equal(t, -1, concreteErr.VersionNumber)

	// Create a version in the fs backend
	data1Hash := backend.ComputeSHA256Hash(test.Data1)
	_, err = fsBackend.CreateOrUpdateModelVersion(
		"foo",
		backend.VersionArgs{Archived: true, DataHash: data1Hash, Data: test.Data1},
	)
	assert.NoError(t, err)

	versionInfo, err := b.RetrieveModelVersionInfo("foo", -1)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(versionInfo.VersionNumber))
	assert.Equal(t, data1Hash, versionInfo.DataHash)
}
