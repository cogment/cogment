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

//go:build cgo && windows
// +build cgo,windows

package wrapper

/*
	#include <windows.h>
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

// Loaded dynamic library datastructre (windows version)
//
// Pretty much generic but is only used and tested for the purpose of the orchestrator
type dynamicLibrary struct {
	libLocalPath string
	hModule      C.HINSTANCE
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

	tmpF.Close()

	pathCStr := C.CString(localPath)
	defer C.free(unsafe.Pointer(pathCStr))
	hModule := C.LoadLibrary(pathCStr)

	if hModule == nil {
		return nil, fmt.Errorf("unable to load dynamic library from %q", localPath)
	}

	return &dynamicLibrary{
		libLocalPath: localPath,
		hModule:      hModule,
	}, nil
}

func (dl *dynamicLibrary) getSymbol(symbol string) (unsafe.Pointer, error) {
	symbolCStr := C.CString(symbol)
	defer C.free(unsafe.Pointer(symbolCStr))
	hProc := C.GetProcAddress(dl.hModule, symbolCStr)
	if hProc == nil {
		return nil, fmt.Errorf("unable to get symbol %q from dynamic library loaded from %q", symbol, dl.libLocalPath)
	}
	return unsafe.Pointer(hProc), nil
}

func (dl *dynamicLibrary) destroy() error {
	if r := C.FreeLibrary(dl.hModule); r == 0 {
		return fmt.Errorf("unable to close the handle to dynamic library loaded from %q", dl.libLocalPath)
	}
	err := os.Remove(dl.libLocalPath)
	if err != nil {
		return err
	}
	return nil
}
