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
	"fmt"
)

const multplier = 1000

var units = []string{"kB", "MB", "GB", "TB", "PB", "EB"}

// Inspired by https://yourbasic.org/golang/formatting-byte-size-to-human-readable-format/
func FormatBytes(bytes int) string {
	if bytes < multplier {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := multplier, 0
	for n := bytes / multplier; n >= multplier; n /= multplier {
		div *= multplier
		exp++
	}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}
