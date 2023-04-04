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

//go:build cgo && (linux || darwin)
// +build cgo
// +build linux darwin

package wrapper

/*
	#cgo LDFLAGS: -ldl

	#include <dlfcn.h>
	#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"io"
	"os"
	"path"
	"unsafe"

	"github.com/markbates/pkger/pkging"
)

// Loaded dynamic library datastructre (linux / macos version)
//
// Pretty much generic but is only used and tested for the purpose of the orchestrator
type dynamicLibrary struct {
	libLocalPath string
	libPtr       unsafe.Pointer
}

func newDynamicLibrary(libraryName string, libraryFile pkging.File) (*dynamicLibrary, error) {
	tmpDir, err := os.MkdirTemp("", "cogment")
	if err != nil {
		return nil, err
	}

	localPath := path.Join(tmpDir, libraryName)
	tmpF, err := os.Create(localPath)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(tmpF, libraryFile)
	if err != nil {
		return nil, err
	}

	pathCStr := C.CString(localPath)
	defer C.free(unsafe.Pointer(pathCStr))
	libPtr := C.dlopen(pathCStr, C.RTLD_LAZY)
	if libPtr == nil {
		errStr := C.GoString(C.dlerror())
		return nil, fmt.Errorf("unable to load dynamic library from %q: %s", localPath, errStr)
	}

	return &dynamicLibrary{
		libLocalPath: localPath,
		libPtr:       libPtr,
	}, nil
}

func (dl *dynamicLibrary) getSymbol(symbol string) (unsafe.Pointer, error) {
	symbolCStr := C.CString(symbol)
	defer C.free(unsafe.Pointer(symbolCStr))
	symbolPtr := C.dlsym(dl.libPtr, symbolCStr)
	if symbolPtr == nil {
		errStr := C.GoString(C.dlerror())
		return nil, fmt.Errorf(
			"unable to get symbol %q from dynamic library loaded from %q: %s",
			symbol,
			dl.libLocalPath,
			errStr,
		)
	}
	return symbolPtr, nil
}

func (dl *dynamicLibrary) destroy() error {
	if r := C.dlclose(unsafe.Pointer(dl.libPtr)); r != 0 {
		errStr := C.GoString(C.dlerror())
		return fmt.Errorf("unable to close the handle to dynamic library loaded from %q: %s", dl.libLocalPath, errStr)
	}
	err := os.Remove(dl.libLocalPath)
	if err != nil {
		return err
	}
	return nil
}
