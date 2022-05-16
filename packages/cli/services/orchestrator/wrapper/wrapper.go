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

package wrapper

import "github.com/cogment/cogment/services/utils"

// Orchestrator wrapper interface
type Wrapper interface {
	Destroy() error

	SetLifecyclePort(port uint) error
	SetActorPort(port uint) error
	SetDefaultParamsFile(path string) error
	AddDirectoryServicesEndpoint(endpoint string) error
	AddPretrialHooksEndpoint(endpoint string) error
	SetPrometheusPort(port uint) error
	SetStatusListener(listener utils.StatusListener) error
	SetPrivateKeyFile(path string) error
	SetRootCertificateFile(path string) error
	SetTrustChainFile(path string) error
	SetGarbageCollectorFrequency(frequency uint) error

	Start() error
	Wait() error
	Shutdown() error
}
