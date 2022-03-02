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

import "fmt"

type Status int

const (
	StatusInitializating Status = iota
	StatusReady
	StatusTerminated
	StatusUnknown
)

func (s Status) String() string {
	return [...]string{"initializing", "ready", "terminated", "unknown"}[s]
}

func (s Status) ToStatusChar() (string, error) {
	if s == StatusUnknown {
		return "", fmt.Errorf("unknown status")
	}
	return [...]string{"I", "R", "T"}[s], nil
}

type StatusListener func(Status)
