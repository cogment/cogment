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

package launcher

import (
	"fmt"

	"github.com/cogment/cogment/utils"
)

// Special predefined substitution strings
const specialPrefix = "__"
const allCliArgsName = "__ALL_ARGS"
const nbCliArgsName = "__NB_ARGS"

type parseDict struct {
	Dict         map[string]string
	reservedName utils.Set
	NbArgs       int
}

func makeParseDict() parseDict {
	var pd parseDict
	pd.Dict = make(map[string]string)

	pd.reservedName = utils.MakeSet()
	pd.reservedName.Add(allCliArgsName)
	pd.Dict[allCliArgsName] = ""
	pd.reservedName.Add(nbCliArgsName)
	pd.Dict[nbCliArgsName] = "0"
	pd.NbArgs = 0

	for argNo := 1; argNo <= 9; argNo++ { // 1-9 are always defined, there is no 0
		argName := fmt.Sprintf("%s%d", specialPrefix, argNo)
		pd.reservedName.Add(argName)
		pd.Dict[argName] = ""
	}

	return pd
}

func (pd *parseDict) Copy() parseDict {
	var newPd parseDict
	newPd.Dict = utils.CopyStrMap(pd.Dict)
	newPd.reservedName = pd.reservedName.Copy()
	newPd.NbArgs = pd.NbArgs
	return newPd
}

func (pd *parseDict) Has(name string) bool {
	_, in := pd.Dict[name]
	return in
}

// Can only add args in order, starting from 1
func (pd *parseDict) AddArg(argNo int, val string) bool {
	if pd.NbArgs+1 != argNo {
		return false
	}

	argName := fmt.Sprintf("%s%d", specialPrefix, argNo)
	pd.reservedName.Add(argName)
	pd.Dict[argName] = val

	allArgs := pd.Dict[allCliArgsName]
	if argNo > 1 {
		allArgs += " "
	}
	allArgs += val
	pd.Dict[allCliArgsName] = allArgs

	pd.NbArgs++
	pd.Dict[nbCliArgsName] = fmt.Sprintf("%d", pd.NbArgs)

	return true
}

func (pd *parseDict) GetArg(argNo int) string {
	argName := fmt.Sprintf("%s%d", specialPrefix, argNo)
	return pd.Dict[argName]
}

func (pd *parseDict) Add(name string, val string) bool {
	if !pd.reservedName.Has(name) {
		pd.Dict[name] = val
		return true
	}

	return false
}

func (pd *parseDict) Get(name string) string {
	return pd.Dict[name]
}
