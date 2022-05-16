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

//go:build cgo && darwin
// +build cgo,darwin

package wrapper

import (
	"github.com/markbates/pkger"
)

// macos specific version of the orchestrator wrapper creation
func NewWrapper() (Wrapper, error) {
	// Tells pkger to embed the file
	f, err := pkger.Open("/services/orchestrator/lib/liborchestrator.dylib")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return newWrapperFromLibrary("liborchestrator.dylib", f)
}
