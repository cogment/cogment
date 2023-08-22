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

package grpcservers

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cogment/cogment/utils"
)

const (
	tickPeriod        = 100 * time.Millisecond
	dbScanPeriod      = 5 * time.Second
	dbScanPeriodTicks = int(dbScanPeriod / tickPeriod)

	healthCheckPeriod     = 30 * time.Second
	maxFailedHealthChecks = 3

	tcpCheckTimeout     = 5 * time.Second
	cogmentCheckTimeout = 2 * time.Second
)

func (ds *DirectoryServer) PeriodicHealthCheck() (func(), error) {
	running := true
	cancelChecks := func() {
		running = false
	}

	go func() {
		tickCount := 0

		log.Debug("Initial directory scan to check health of persisted entries")
		ds.nowHealthCheck(&running, true)

		for running {
			if tickCount < dbScanPeriodTicks {
				time.Sleep(tickPeriod)
				tickCount++
			} else {
				tickCount = 0

				log.Debug("Periodic directory scan to check health of entries")
				ds.nowHealthCheck(&running, false)
			}
		}
	}()

	return cancelChecks, nil
}

func (ds *DirectoryServer) nowHealthCheck(running *bool, immediateRemoval bool) {
	ids, err := ds.db.SelectAllIDs()
	if err != nil {
		log.Error("Health check failed; could not retrieve IDs")
		return
	}

	now := utils.Timestamp()
	for _, id := range ids {
		if !*running {
			break
		}

		recordID := id
		go func() {
			record, err := ds.db.SelectByID(recordID)
			if err != nil {
				// Id was removed in the interim
				return
			}

			if (now - record.LastHealthCheckTimestamp) < uint64(healthCheckPeriod.Nanoseconds()) {
				return
			}

			healthy := healthCheck(record, now)
			if !healthy && (immediateRemoval || record.NbFailedHealthChecks >= maxFailedHealthChecks) {
				ds.db.Delete(recordID)

				if immediateRemoval {
					log.WithField("service_id", recordID).Info("Service removed after failing initial health check")
				} else {
					log.WithField("service_id", recordID).Info("Service removed after failing multiple health checks")
				}
			}
		}()
	}
}

func healthCheck(record *DbRecord, now uint64) bool {
	record.LastHealthCheckTimestamp = now

	if record.Permanent {
		record.NbFailedHealthChecks = 0
		return true
	}

	if record.Endpoint.Protocol != cogmentAPI.ServiceEndpoint_GRPC &&
		record.Endpoint.Protocol != cogmentAPI.ServiceEndpoint_GRPC_SSL {
		// The endpoint is not a network resource
		record.NbFailedHealthChecks = 0
		return true
	}

	// We don't check cogment health if not a cogment service (or if SSL required), only tcp health.
	// TODO: Should we really check non cogment services? They could be UDP, or something else completely!
	// TODO: The directory could be set with SSL certificates, then we could try to check cogment SSL services
	if record.Details.Type > cogmentAPI.ServiceType_MODEL_REGISTRY_SERVICE ||
		record.Endpoint.Protocol == cogmentAPI.ServiceEndpoint_GRPC_SSL {
		var err error
		timeout := tcpCheckTimeout
		for index := 0; index < 3; index++ {
			err = tcpCheck(record.Endpoint.Host, record.Endpoint.Port, timeout)
			if err == nil {
				break
			}
			log.Debug("Failed TCP transient health check for ID [", record.Sid, "] - ", err.Error())
			timeout *= 2
		}
		if err != nil {
			record.NbFailedHealthChecks++
			log.Debug("Failed TCP health check [", record.NbFailedHealthChecks, "] for ID [", record.Sid, "]")
			return false
		}
	} else {
		var err error
		timeout := cogmentCheckTimeout
		for index := 0; index < 3; index++ {
			err = cogmentCheck(record.Details.Type, record.Endpoint.Host, record.Endpoint.Port, timeout)
			if err == nil {
				break
			}
			log.Debug("Failed Cogment transient health check for ID [", record.Sid, "] - ", err.Error())
			timeout *= 2
		}
		if err != nil {
			record.NbFailedHealthChecks++
			log.Debug("Failed Cogment health check [", record.NbFailedHealthChecks, "] for ID [", record.Sid, "]")
			return false
		}
	}

	record.NbFailedHealthChecks = 0
	return true
}

func tcpCheck(host string, port uint32, timeout time.Duration) error {
	address := fmt.Sprintf("%s:%d", host, port)

	var dialer net.Dialer
	ctx, cancelContext := context.WithTimeout(context.Background(), timeout)
	defer cancelContext()

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()

	// TODO: Review with more use cases -> should we send data and check for received data (as done here)?
	optionsRequest := []byte("OPTIONS * HTTP/1.1\r\n\r\n") // "Arbitrary" string (since it may not be an HTTP server)
	_, err = conn.Write(optionsRequest)
	if err != nil {
		return err
	}

	response := make([]byte, 256)
	count, err := conn.Read(response)
	if err == nil || err == io.EOF {
		log.Debug("Response (", count, ") '", string(response[:count]), "' ", response[:count])
		return nil
	}

	return err
}

func cogmentCheck(serviceType cogmentAPI.ServiceType, host string, port uint32, timeout time.Duration) error {
	address := fmt.Sprintf("%s:%d", host, port)

	ctx, cancelContext := context.WithTimeout(context.Background(), timeout)
	defer cancelContext()

	connection, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return err
	}
	defer connection.Close()

	method := ""
	switch serviceType {
	case cogmentAPI.ServiceType_TRIAL_LIFE_CYCLE_SERVICE:
		method = "/cogmentAPI.TrialLifecycleSP/Version"
	case cogmentAPI.ServiceType_CLIENT_ACTOR_CONNECTION_SERVICE:
		method = "/cogmentAPI.ClientActorSP/Version"
	case cogmentAPI.ServiceType_ACTOR_SERVICE:
		method = "/cogmentAPI.ServiceActorSP/Version"
	case cogmentAPI.ServiceType_ENVIRONMENT_SERVICE:
		method = "/cogmentAPI.EnvironmentSP/Version"
	case cogmentAPI.ServiceType_PRE_HOOK_SERVICE:
		method = "/cogmentAPI.TrialHooksSP/Version"
	case cogmentAPI.ServiceType_DATALOG_SERVICE:
		method = "/cogmentAPI.DatalogSP/Version"
	case cogmentAPI.ServiceType_DATASTORE_SERVICE:
		method = "/cogmentAPI.TrialDatastoreSP/Version"
	case cogmentAPI.ServiceType_MODEL_REGISTRY_SERVICE:
		method = "/cogmentAPI.ModelRegistrySP/Version"
	default:
		log.Error("Unknown gRPC service type [", serviceType, "]")
		method = "/?/Version"
	}

	// We could use two generic empty protobuf instead
	in := cogmentAPI.VersionRequest{}
	out := cogmentAPI.VersionInfo{}
	err = connection.Invoke(ctx, method, &in, &out, grpc.WaitForReady(false))
	return err
}
