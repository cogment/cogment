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

//go:build cgo
// +build cgo

package wrapper

/*
	#include <stdlib.h>
	#include <time.h>

	typedef void* (*OptionsCreateFun)();
	void* call_options_create_fun(void* f) {
		return ((OptionsCreateFun) f)();
	}

	typedef void (*OptionsDestroyFun)(void*);
	void call_options_destroy_fun(void* f, void* options) {
		return ((OptionsDestroyFun) f)(options);
	}

	typedef void (*SetUintOptionsFun) (void* options, unsigned int value);
 	void call_set_uint_options_fun(void* f, void* options, unsigned int value) {
		return ((SetUintOptionsFun) f)(options, value);
	}

	typedef void (*SetStringOptionsFun) (void* options, char* value);
 	void call_set_string_options_fun(void* f, void* options, char* value) {
		return ((SetStringOptionsFun) f)(options, value);
	}

	typedef void (*CogmentOrchestratorLogger)(void*, char*, int, time_t, size_t, char*, int, char*,
                                            char*);
	typedef void (*SetLoggingOptionsFun)(void* options, char* level, void* ctx, CogmentOrchestratorLogger logger);
	void call_set_logging_options_fun(void* f, void* options, char* level, void* ctx, void* logger) {
		return ((SetLoggingOptionsFun) f)(options, level, ctx, (CogmentOrchestratorLogger) logger);
	}

	typedef void (*CogmentOrchestratorStatusListener)(void* ctx, int status);
	typedef void (*SetStatusListenerOptionsFun)(void* options, void* ctx, CogmentOrchestratorStatusListener listener);
	void call_set_status_listener_options_fun(void* f, void* options, void* ctx, void* listener) {
		return ((SetStatusListenerOptionsFun) f)(options, ctx, (CogmentOrchestratorStatusListener) listener);
	}

	typedef void* (*OrchestratorCreateFun)(void* options);
	void* call_orchestrator_create_fun(void* f, void* options) {
		return ((OrchestratorCreateFun) f)(options);
	}

	typedef void (*OrchestratorDestroyFun)(void* orchestrator);
	void call_orchestrator_destroy_fun(void* f, void* orchestrator) {
		return ((OrchestratorDestroyFun) f)(orchestrator);
	}

	typedef int (*OrchestratorWaitFun)(void*);
	int call_orchestrator_wait_fun(void* f, void* orchestrator) {
		return ((OrchestratorWaitFun) f)(orchestrator);
	}

	typedef void (*OrchestratorFun)(void*);
	void call_orchestrator_fun(void* f, void* orchestrator) {
		((OrchestratorFun) f)(orchestrator);
	}
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/cogment/cogment/services/utils"
	"github.com/markbates/pkger/pkging"
	"github.com/sirupsen/logrus"
)

// Wapper around the dynamically loaded orchestrator (requires CGO)
type wrapper struct {
	dynamicLibrary
	optionsPtr           unsafe.Pointer
	orchestratorPtr      unsafe.Pointer
	statusListenerHandle uintptr
}

func newWrapperFromLibrary(libraryName string, libraryFile pkging.File) (Wrapper, error) {
	dynamicLibrary, err := newDynamicLibrary(libraryName, libraryFile)
	if err != nil {
		return nil, err
	}

	log.WithField("path", dynamicLibrary.libLocalPath).Debug("orchestrator service dynamic library unpacked and loaded")

	w := &wrapper{
		dynamicLibrary:       *dynamicLibrary,
		statusListenerHandle: 0,
	}

	createFun, err := w.getSymbol("cogment_orchestrator_options_create")
	if err != nil {
		return nil, err
	}
	w.optionsPtr = C.call_options_create_fun(createFun)
	if w.optionsPtr == nil {
		return nil, fmt.Errorf("unable to create the orchestrator options datastructure")
	}

	setLoggingFun, err := w.getSymbol("cogment_orchestrator_options_set_logging")
	if err != nil {
		return nil, err
	}
	var spdlogLevel string
	switch log.Logger.GetLevel() {
	case logrus.TraceLevel:
		spdlogLevel = "trace"
	case logrus.DebugLevel:
		spdlogLevel = "debug"
	case logrus.InfoLevel:
		spdlogLevel = "info"
	case logrus.WarnLevel:
		spdlogLevel = "warning"
	case logrus.ErrorLevel:
		spdlogLevel = "error"
	case logrus.FatalLevel:
		spdlogLevel = "critical"
	case logrus.PanicLevel:
		// Panic level is meaningless in C++
		spdlogLevel = "off"
	}
	var levelCStr = C.CString(spdlogLevel)
	defer C.free(unsafe.Pointer(levelCStr))
	C.call_set_logging_options_fun(setLoggingFun, w.optionsPtr, levelCStr, nil, c_log_callback)

	return w, nil
}

func (w *wrapper) Destroy() error {
	if w.orchestratorPtr != nil {
		destroyOrchestratorFun, err := w.getSymbol("cogment_orchestrator_destroy")
		if err != nil {
			return err
		}
		C.call_orchestrator_destroy_fun(destroyOrchestratorFun, w.orchestratorPtr)
		w.orchestratorPtr = nil
	}

	destroyOptionsFun, err := w.getSymbol("cogment_orchestrator_options_destroy")
	if err != nil {
		return err
	}
	C.call_options_destroy_fun(destroyOptionsFun, w.optionsPtr)
	w.optionsPtr = nil

	if w.statusListenerHandle != 0 {
		statusListenerRegistrySingleton.unregister(w.statusListenerHandle)
	}

	err = w.dynamicLibrary.destroy()
	if err != nil {
		return err
	}
	return nil
}

func (w *wrapper) setUintOption(symbol string, value uint) error {
	fun, err := w.getSymbol(symbol)
	if err != nil {
		return err
	}

	C.call_set_uint_options_fun(fun, w.optionsPtr, C.uint(value))
	return nil
}

func (w *wrapper) setStringOption(symbol string, value string) error {
	fun, err := w.getSymbol(symbol)
	if err != nil {
		return err
	}

	valueCStr := C.CString(value)
	defer C.free(unsafe.Pointer(valueCStr))

	C.call_set_string_options_fun(fun, w.optionsPtr, valueCStr)
	return nil
}

func (w *wrapper) SetLifecyclePort(port uint) error {
	return w.setUintOption("cogment_orchestrator_options_set_lifecycle_port", port)
}

func (w *wrapper) SetActorPort(port uint) error {
	return w.setUintOption("cogment_orchestrator_options_set_actor_port", port)
}

func (w *wrapper) SetDefaultParamsFile(path string) error {
	return w.setStringOption("cogment_orchestrator_options_set_default_params_file", path)
}

func (w *wrapper) AddDirectoryServicesEndpoint(endpoint string) error {
	return w.setStringOption("cogment_orchestrator_options_add_directory_service", endpoint)
}

func (w *wrapper) SetDirectoryAuthToken(token string) error {
	return w.setStringOption("cogment_orchestrator_options_set_directory_auth_token", token)
}

func (w *wrapper) SetDirectoryAutoRegister(autoRegister uint) error {
	return w.setUintOption("cogment_orchestrator_options_set_auto_registration", autoRegister)
}

func (w *wrapper) SetDirectoryRegisterHost(host string) error {
	return w.setStringOption("cogment_orchestrator_options_set_directory_register_host", host)
}

func (w *wrapper) SetDirectoryRegisterProps(props string) error {
	return w.setStringOption("cogment_orchestrator_options_set_directory_properties", props)
}

func (w *wrapper) AddPretrialHooksEndpoint(endpoint string) error {
	return w.setStringOption("cogment_orchestrator_options_add_pretrial_hook", endpoint)
}

func (w *wrapper) SetPrometheusPort(port uint) error {
	return w.setUintOption("cogment_orchestrator_options_set_prometheus_port", port)
}

func (w *wrapper) SetStatusListener(listener utils.StatusListener) error {
	setStatusListenerFun, err := w.getSymbol("cogment_orchestrator_options_set_status_listener")
	if err != nil {
		return err
	}
	if w.statusListenerHandle != 0 {
		statusListenerRegistrySingleton.unregister(w.statusListenerHandle)
	}
	handle := statusListenerRegistrySingleton.register(listener)
	C.call_set_status_listener_options_fun(setStatusListenerFun, w.optionsPtr, unsafe.Pointer(handle), c_status_listener_callback)

	return nil
}

func (w *wrapper) SetPrivateKeyFile(path string) error {
	return w.setStringOption("cogment_orchestrator_options_set_private_key_file", path)
}

func (w *wrapper) SetRootCertificateFile(path string) error {
	return w.setStringOption("cogment_orchestrator_options_set_root_certificate_file", path)
}

func (w *wrapper) SetTrustChainFile(path string) error {
	return w.setStringOption("cogment_orchestrator_options_trust_chain_file", path)
}

func (w *wrapper) SetGarbageCollectorFrequency(frequency uint) error {
	return w.setUintOption("cogment_orchestrator_options_garbage_collector_frequency", frequency)
}

func (w *wrapper) Start() error {
	createAndStartPtr, err := w.getSymbol("cogment_orchestrator_create_and_start")
	if err != nil {
		return err
	}

	w.orchestratorPtr = C.call_orchestrator_create_fun(createAndStartPtr, w.optionsPtr)
	if w.orchestratorPtr == nil {
		return fmt.Errorf("unable to create the orchestrator datastructure")
	}
	return nil
}

func (w *wrapper) Wait() error {
	if w.orchestratorPtr == nil {
		return fmt.Errorf("orchestrator wasn't started yet")
	}

	waitPtr, err := w.getSymbol("cogment_orchestrator_wait_for_termination")
	if err != nil {
		return err
	}

	log.Debug("waiting for the orchestrator service to finish...")
	exitCode := C.call_orchestrator_wait_fun(waitPtr, w.orchestratorPtr)
	log.WithField("exitCode", exitCode).Debug("orchestrator service finished")
	if exitCode < 0 {
		return fmt.Errorf("Unexpected error while waiting for the orchestrator, exit_code=%d", exitCode)
	}
	return nil
}

func (w *wrapper) Shutdown() error {
	if w.orchestratorPtr == nil {
		return fmt.Errorf("orchestrator wasn't started yet")
	}

	shutdownPtr, err := w.getSymbol("cogment_orchestrator_shutdown")
	if err != nil {
		return err
	}

	log.Info("shutting down the orchestrator...")
	C.call_orchestrator_fun(shutdownPtr, w.orchestratorPtr)
	return nil
}
