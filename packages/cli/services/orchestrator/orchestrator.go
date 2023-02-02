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

package orchestrator

import (
	"context"
	"fmt"
	"os"

	"github.com/cogment/cogment/services/orchestrator/proxy"
	"github.com/cogment/cogment/services/orchestrator/wrapper"
	"github.com/cogment/cogment/services/utils"
	baseUtils "github.com/cogment/cogment/utils"
	"golang.org/x/sync/errgroup"
)

type Options struct {
	utils.DirectoryRegistrationOptions
	LifecyclePort             uint
	ActorPort                 uint
	ActorWebPort              uint
	ParamsFile                string
	PretrialHooksEndpoints    []string
	PrometheusPort            uint
	StatusFile                string
	PrivateKeyFile            string
	RootCertificateFile       string
	TrustChainFile            string
	GarbageCollectorFrequency uint
	DirectoryAutoRegister     bool
}

var DefaultOptions = Options{
	DirectoryRegistrationOptions: utils.DefaultDirectoryRegistrationOptions,
	LifecyclePort:                9000,
	ActorPort:                    9000,
	ActorWebPort:                 0,
	ParamsFile:                   "",
	PrometheusPort:               0,
	StatusFile:                   "",
	PrivateKeyFile:               "",
	RootCertificateFile:          "",
	TrustChainFile:               "",
	GarbageCollectorFrequency:    10,
	DirectoryAutoRegister:        true,
}

func Run(ctx context.Context, options Options) error {
	g, ctx := errgroup.WithContext(ctx)

	orchestratorReady := make(chan struct{})
	orchestratorStatus := make(chan utils.Status)

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case status := <-orchestratorStatus:
				log.Debugf("orchestrator is now %v", status)
				// If now ready pass the information to the dedicated channel
				if status == utils.StatusReady {
					go func() {
						orchestratorReady <- struct{}{}
					}()
				}
				if options.StatusFile != "" {
					file, err := os.OpenFile(options.StatusFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

					if err != nil {
						return fmt.Errorf("unable to open status file %q: %w", options.StatusFile, err)
					}

					defer file.Close()

					statusChar, err := status.ToStatusChar()
					if err != nil {
						return err
					}

					_, err = file.WriteString(statusChar)
					if err != nil {
						return fmt.Errorf("unable to write to status file %q: %w", options.StatusFile, err)
					}
				}
			}
		}
	})

	g.Go(func() error {
		return runOrchestrator(ctx, options, func(status utils.Status) {
			go func() {
				orchestratorStatus <- status
			}()
		})
	})

	if options.ActorWebPort > 0 {
		proxyOptions := proxy.Options{
			BackendPort: options.ActorPort,
			WebPort:     options.ActorWebPort,
		}
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-orchestratorReady: // Waiting until the orchestrator is ready
				return proxy.Run(ctx, proxyOptions)
			}
		})
	}

	return g.Wait()
}

func runOrchestrator(ctx context.Context, options Options, statusListener utils.StatusListener) error {
	w, err := wrapper.NewWrapper()
	if err != nil {
		return err
	}
	defer func() {
		err := w.Destroy()
		if err != nil {
			log.WithField("error", err).Error("Error while destroying the orchestrator wrapper")
		}
	}()
	err = w.SetStatusListener(statusListener)
	if err != nil {
		return err
	}

	checker := utils.NewNetChecker()

	err = checker.TCPPort(uint16(options.LifecyclePort))
	if err != nil {
		return fmt.Errorf("Lifecycle port [%d] unavailable: %w", options.LifecyclePort, err)
	}
	err = w.SetLifecyclePort(options.LifecyclePort)
	if err != nil {
		return err
	}

	err = checker.TCPPort(uint16(options.ActorPort))
	if err != nil {
		return fmt.Errorf("Actor port [%d] unavailable: %w", options.ActorPort, err)
	}
	err = w.SetActorPort(options.ActorPort)
	if err != nil {
		return err
	}

	err = w.SetDefaultParamsFile(options.ParamsFile)
	if err != nil {
		return err
	}
	for _, pretrialHook := range options.PretrialHooksEndpoints {
		err = w.AddPretrialHooksEndpoint(pretrialHook)
		if err != nil {
			return err
		}
	}
	err = w.AddDirectoryServicesEndpoint(options.DirectoryEndpoint)
	if err != nil {
		return err
	}
	err = w.SetDirectoryAuthToken(options.DirectoryAuthToken)
	if err != nil {
		return err
	}
	if options.DirectoryAutoRegister {
		err = w.SetDirectoryAutoRegister(1)
	} else {
		err = w.SetDirectoryAutoRegister(0)
	}
	if err != nil {
		return err
	}
	err = w.SetDirectoryRegisterHost(options.DirectoryRegistrationHost)
	if err != nil {
		return err
	}
	err = w.SetDirectoryRegisterProps(baseUtils.FormatStringToString(options.DirectoryRegistrationProperties))
	if err != nil {
		return err
	}

	if options.PrometheusPort > 0 {
		err = checker.TCPPort(uint16(options.PrometheusPort))
		if err != nil {
			return fmt.Errorf("Prometheus port [%d] unavailable: %w", options.PrometheusPort, err)
		}
	}
	err = w.SetPrometheusPort(options.PrometheusPort)
	if err != nil {
		return err
	}
	err = w.SetPrivateKeyFile(options.PrivateKeyFile)
	if err != nil {
		return err
	}
	err = w.SetRootCertificateFile(options.RootCertificateFile)
	if err != nil {
		return err
	}
	err = w.SetTrustChainFile(options.TrustChainFile)
	if err != nil {
		return err
	}
	err = w.SetGarbageCollectorFrequency(options.GarbageCollectorFrequency)
	if err != nil {
		return err
	}

	err = w.Start()
	if err != nil {
		return err
	}

	orchestratorRes := make(chan error)
	go func() {
		orchestratorRes <- w.Wait()
	}()

	select {
	case <-ctx.Done():
		err := w.Shutdown()
		if err != nil {
			return fmt.Errorf("unable to shutdown the orchestrator: %w", err)
		}
		err = <-orchestratorRes
		if err != nil {
			return fmt.Errorf("error while waiting for the orchestrator to shutdown: %w", err)
		}
		return ctx.Err()
	case err := <-orchestratorRes:
		return err
	}
}
