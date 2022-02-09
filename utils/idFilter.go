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

package utils

type IDFilter map[string]struct{}

func NewIDFilter(selectedIDs []string) IDFilter {
	set := make(IDFilter)
	for _, id := range selectedIDs {
		set[id] = struct{}{}
	}
	return set
}

func (f *IDFilter) SelectsAll() bool {
	return len(*f) == 0
}

func (f *IDFilter) Selects(id string) bool {
	if len(*f) == 0 {
		return true
	}
	_, isSelected := (*f)[id]
	return isSelected
}
