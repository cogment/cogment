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

package utils

import (
	"math/rand"
	"strings"
	"time"
)

// RFC4648-5 base64 URL/file safe character pool
var base64Pool = "ABCDEFGHIJKLMNOPQRSTUVWZYZabcdefghijklmnopqrstuvwzyz0123456789-_"

func init() {
	rand.Seed(time.Now().UnixNano()) //nolint
}

func Timestamp() uint64 {
	return uint64(time.Now().UnixNano())
}

func RandomUint() uint64 {
	// For now, 63 bits is enough
	return uint64(rand.Int63())
}

func RandomString(length int, sources ...string) string {
	var pool *string
	if len(sources) == 0 {
		pool = &base64Pool
	} else if len(sources) == 1 {
		pool = &(sources[0])
	} else {
		*pool = ""
		for _, partialSource := range sources {
			*pool += partialSource
		}
	}

	var result strings.Builder
	for count := 0; count < length; count++ {
		index := rand.Intn(len(*pool))
		result.WriteByte((*pool)[index])
	}

	return result.String()
}

// Copy string slice in a new array
func CopyStrSlice(src []string) []string {
	result := make([]string, len(src))
	copy(result, src)
	return result
}

func CopyStrMap(src map[string]string) map[string]string {
	result := make(map[string]string, len(src))
	for name, value := range src {
		result[name] = value
	}

	return result
}
