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

type Set struct {
	Data map[interface{}]bool
}

func MakeSet() Set {
	var st Set
	st.Data = make(map[interface{}]bool)
	return st
}

func (st *Set) Has(val interface{}) bool {
	return st.Data[val]
}

func (st *Set) Add(val interface{}) {
	st.Data[val] = true
}

func (st *Set) Remove(val interface{}) {
	delete(st.Data, val)
}

func (st *Set) Size() int {
	return len(st.Data)
}

func (st *Set) Clear() {
	for item := range st.Data {
		delete(st.Data, item)
	}
}

func (st *Set) Copy() Set {
	var newSt Set
	newSt.Data = make(map[interface{}]bool, len(st.Data))
	for name := range st.Data {
		newSt.Data[name] = true
	}

	return newSt
}
