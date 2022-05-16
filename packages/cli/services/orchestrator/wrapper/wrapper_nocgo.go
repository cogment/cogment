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

//go:build !cgo
// +build !cgo

package wrapper

import (
	"fmt"

	"github.com/cogment/cogment/services/utils"
)

// "Mock" version of the orchestrator wrapper for non-cgo builds
type unsupportedWrapper struct {
}

func NewWrapper() (Wrapper, error) {
	return &unsupportedWrapper{}, fmt.Errorf("Orchestrator service not supported")
}

func (w *unsupportedWrapper) Destroy() error {
	return nil
}

func (w *unsupportedWrapper) SetLifecyclePort(port uint) error {
	return fmt.Errorf("Orchestrator service not supported")
}

func (w *unsupportedWrapper) SetActorPort(port uint) error {
	return fmt.Errorf("Orchestrator service not supported")
}

func (w *unsupportedWrapper) SetDefaultParamsFile(path string) error {
	return fmt.Errorf("Orchestrator service not supported")
}

func (w *unsupportedWrapper) AddDirectoryServicesEndpoint(endpoint string) error {
	return fmt.Errorf("Orchestrator service not supported")
}

func (w *unsupportedWrapper) AddPretrialHooksEndpoint(endpoint string) error {
	return fmt.Errorf("Orchestrator service not supported")
}

func (w *unsupportedWrapper) SetPrometheusPort(port uint) error {
	return fmt.Errorf("Orchestrator service not supported")
}

func (w *unsupportedWrapper) SetStatusListener(listener utils.StatusListener) error {
	return fmt.Errorf("Orchestrator service not supported")
}

func (w *unsupportedWrapper) SetPrivateKeyFile(path string) error {
	return fmt.Errorf("Orchestrator service not supported")
}

func (w *unsupportedWrapper) SetRootCertificateFile(path string) error {
	return fmt.Errorf("Orchestrator service not supported")
}

func (w *unsupportedWrapper) SetTrustChainFile(path string) error {
	return fmt.Errorf("Orchestrator service not supported")
}

func (w *unsupportedWrapper) SetGarbageCollectorFrequency(frequency uint) error {
	return fmt.Errorf("Orchestrator service not supported")
}

func (w *unsupportedWrapper) Start() error {
	return fmt.Errorf("Orchestrator service not supported")
}

func (w *unsupportedWrapper) Wait() error {
	return fmt.Errorf("Orchestrator service not supported")
}

func (w *unsupportedWrapper) Shutdown() error {
	return fmt.Errorf("Orchestrator service not supported")
}
