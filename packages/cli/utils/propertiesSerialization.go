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
	"strings"
)

const (
	propertiesKvSeparator string = "="
	propertiesSeparator   string = ","
)

// FormatProperties formats a `map[string]string` to a "property string" format (e.g. "key1=value1,key2=value2')
func FormatProperties(properties map[string]string) string {
	str := ""
	for k, v := range properties {
		if len(str) > 0 {
			str += propertiesSeparator
		}
		str += k
		if v != "" {
			str += propertiesKvSeparator + v
		}

	}
	return str
}

// ParseProperties parses a "properties string" format to a `map[string]string`
func ParseProperties(str string) (map[string]string, error) {
	result := make(map[string]string)

	properties := strings.Split(str, propertiesSeparator)
	for _, property := range properties {
		propertyParts := strings.Split(strings.TrimSpace(property), propertiesKvSeparator)
		nbParts := len(propertyParts)

		if nbParts > 2 {
			return nil, fmt.Errorf("Invalid property format [%s]", property)
		}

		name := strings.TrimSpace(propertyParts[0])
		if len(name) == 0 {
			if nbParts == 1 {
				continue
			}
			return nil, fmt.Errorf("Empty property name [%s]", property)
		}

		if nbParts == 2 {
			value := strings.TrimSpace(propertyParts[1])
			result[name] = value
		} else {
			result[name] = ""
		}
	}

	return result, nil
}
