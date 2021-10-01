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

package memoryCache

import (
	"testing"

	"github.com/cogment/cogment-model-registry/backend"
	"github.com/cogment/cogment-model-registry/backend/fs"
	"github.com/cogment/cogment-model-registry/backend/test"
	"github.com/stretchr/testify/assert"
)

func TestSuiteMemoryCacheBackend(t *testing.T) {
	test.RunSuite(t, func() backend.Backend {
		fsBackend, err := fs.CreateBackend(t.TempDir())
		assert.NoError(t, err)

		b, err := CreateBackend(fsBackend)
		assert.NoError(t, err)
		return b
	}, func(b backend.Backend) {
		mcb := b.(*memoryCacheBackend)
		mcb.archive.Destroy()
		mcb.Destroy()
	})
}
