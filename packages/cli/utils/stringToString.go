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

package utils

import (
	"fmt"
	"net/url"
	"strings"
)

// FormatStringToString formats a `map[string]string` to a "property string" format (e.g. "key1=value1,key2=value2')
func FormatStringToString(value map[string]string) string {
	str := ""
	for k, v := range value {
		if len(str) > 0 {
			str += ","
		}
		str += k + "=" + v
	}
	return str
}

// ParseStringToString parses a "property string" format to a `map[string]string`
func ParseStringToString(str string) (map[string]string, error) {
	value := map[string]string{}
	urlQueryStr := strings.ReplaceAll(str, ",", "&")
	parsedURLQuery, err := url.ParseQuery(urlQueryStr)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse \"string to string\" map from [%s], invalid format", str)
	}

	for k, v := range parsedURLQuery {
		value[k] = v[len(v)-1]
	}

	return value, nil
}
