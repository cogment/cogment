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

package constants

const (
	ServiceTypeActor         = "actor"
	ServiceTypeEnvironment   = "environment"
	ServiceTypePrehook       = "prehook"
	ServiceTypeDatalog       = "datalog"
	ServiceTypeLifecycle     = "lifecycle"
	ServiceTypeClientActor   = "actservice"
	ServiceTypeDatastore     = "datastore"
	ServiceTypeModelRegistry = "modelregistry"
	ServiceTypeDirectory     = "directory"

	ServiceTypeOther = "other"

	ServiceTypeAllDesc = "actor, environment, prehook, datalog, lifecycle, actservice" +
		", datastore, modelregistry, directory, other"
)

// This is meant to be used a const (because Go is stupid).
// Do not assume a specific order in this list.
func ServiceTypeAllList() []string {
	return []string{
		"actor",
		"environment",
		"prehook",
		"datalog",
		"lifecycle",
		"actservice",
		"datastore",
		"modelregistry",
		"directory",
		"other",
	}
}
