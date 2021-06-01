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

package api

type Application struct {
	Id               string `json:"id,omitempty"`
	Name             string `json:"name"`
	CreatedAt        int    `json:"created_at,omitempty"`
	LastDeploymentAt int    `json:"last_deployment_at,omitempty"`
}

type ApplicationDetails struct {
	Id          string                   `json:"id,omitempty"`
	CreatedAt   int                      `json:"created_at,omitempty"`
	Status      string                   `json:"status"`
	ErrorReason string                   `json:"error_reason"`
	Services    map[string]ServiceStatus `json:"services"`
}

type ServiceStatus struct {
	Image     string   `json:"image"`
	Endpoints []string `json:"endpoints"`
	Status    string   `json:"status"`
	Errors    []string `json:"errors"`
}
