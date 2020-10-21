// Copyright 2020 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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
	"encoding/json"
	"log"
	"regexp"
	"strings"
)

func PrettyPrint(i interface{}) string {
	s, err := json.MarshalIndent(i, "", "\t")
	if err != nil {
		log.Printf("prettprint.Get err   #%v ", err)
	}
	return string(s)
}

func Snakeify(data string) string {
	space := regexp.MustCompile(`\s+`)
	data = strings.ToLower(data)
	data = space.ReplaceAllString(data, "_")
	return data
}

func Kebabify(data string) string {
	space := regexp.MustCompile(`(\s|_)+`)
	data = strings.ToLower(data)
	data = space.ReplaceAllString(data, "-")
	return data
}

func Pascalify(data string) string {
	space := regexp.MustCompile(`\s+`)
	data = strings.Title(data)
	data = space.ReplaceAllString(data, "")
	return data
}

func Tocaps(data string) string {
	data = strings.ToUpper(data)
	return data
}
