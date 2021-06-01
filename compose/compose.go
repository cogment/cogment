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

package compose

type PortDefinition struct {
	Port   uint16
	Public bool
}

type XCogment struct {
	Image       string
	Command     string
	Entrypoint  string
	Ports       []*PortDefinition
	Environment map[string]string
}

type Service struct {
	Image    string
	Build    interface{}
	XCogment *XCogment `yaml:"x-cogment"`
}

type Manifest struct {
	//Version  string
	Services map[string]Service
}
