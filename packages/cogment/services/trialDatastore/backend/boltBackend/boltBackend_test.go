// Copyright 2022 AI Redefined Inc. <dev+cogment@ai-r.com>
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

package boltBackend

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cogment/cogment/services/trialDatastore/backend"
	"github.com/cogment/cogment/services/trialDatastore/backend/test"
)

func TestSuiteBoltBackend(t *testing.T) {
	test.RunSuite(t, func() backend.Backend {
		// create and open a temporary file
		f, err := os.CreateTemp("", "trial-datastore-bolt-test")
		assert.NoError(t, err)

		// close and remove the temporary file
		defer f.Close()

		b, err := CreateBoltBackend(f.Name())
		assert.NoError(t, err)
		return b
	}, func(b backend.Backend) {
		rb := b.(*boltBackend)

		defer os.Remove(rb.filePath)
		defer rb.Destroy()
	})
}

func BenchmarkBoltBackend(b *testing.B) {
	test.RunBenchmarks(b, func() backend.Backend {
		// create and open a temporary file
		f, err := os.CreateTemp("", "trial-datastore-bolt-benchmark")
		assert.NoError(b, err)

		// close and remove the temporary file
		defer f.Close()

		bck, err := CreateBoltBackend(f.Name())
		assert.NoError(b, err)
		return bck
	}, func(bck backend.Backend) {
		rb := bck.(*boltBackend)

		defer os.Remove(rb.filePath)
		defer rb.Destroy()
	})
}
