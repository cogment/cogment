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

package helper

import (
	"fmt"
	"regexp"
)

// VersionInfo represents information about a package and its version
type VersionInfo struct {
	Package string
	Version string
}

var versionRegex *regexp.Regexp = regexp.MustCompile(`v?([0-9]+(?:\.[0-9]+(?:\.[0-9]+(?:-[a-zA-Z0-9]+)?)?)?)`)

// SanitizeVersion check and sanitize a given version string.
//
// In particular it removes any "v" prefix
func SanitizeVersion(versionString string) (string, error) {
	matches := versionRegex.FindStringSubmatch(versionString)
	if matches == nil {
		return "", fmt.Errorf("Unable to sanitize version string %q is an invalid version", versionString)
	}
	return matches[1], nil
}
