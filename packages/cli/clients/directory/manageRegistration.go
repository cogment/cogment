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

package directory

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/utils/endpoint"
	"github.com/cogment/cogment/version"
	"github.com/sirupsen/logrus"
)

// A RegistrationOptions holds the options for a service to be registered with one or more directories
//
// This struct is designed to be used as a "mixin" in services options.
type RegistrationOptions struct {
	DirectoryEndpoint               endpoint.Endpoint
	DirectoryAuthToken              string
	DirectoryRegistrationHost       string
	DirectoryRegistrationProperties map[string]string
}

var DefaultRegistrationOptions = RegistrationOptions{
	DirectoryEndpoint:               endpoint.Endpoint{},
	DirectoryAuthToken:              "",
	DirectoryRegistrationHost:       "",
	DirectoryRegistrationProperties: map[string]string{},
}

func getOutboundIP() (net.IP, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve outbound ip [%w]", err)
	}
	hostIPs, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve outbound ip [%w]", err)
	}
	for _, hostIP := range hostIPs {
		if hostIP.IsLoopback() {
			continue
		}
		if hostIPv4 := hostIP.To4(); hostIPv4 != nil {
			return hostIPv4, err
		}
	}
	return nil, fmt.Errorf("No valid ipv4 outbound ip found")
}

const serviceDeregisterTimeout = time.Second * 10

func deregisterService(
	log *logrus.Entry,
	directoryEndpoint endpoint.Endpoint,
	directoryAuthToken string,
	serviceID uint64,
	serviceSecret string,
) {
	ctx, cancel := context.WithTimeout(context.Background(), serviceDeregisterTimeout)
	defer cancel()
	directoryClient, err := CreateClient(ctx, directoryEndpoint, directoryAuthToken)
	if err != nil {
		log.WithField("error", err).Debug("Error while deregistering service from the directory")
		return
	}
	log.WithFields(logrus.Fields{
		"serviceID": serviceID,
	}).Debug("Deregistering service from the directory")
	err = directoryClient.Deregister(&api.DeregisterRequest{
		ServiceId: serviceID,
		Secret:    serviceSecret,
	})

	if err == context.DeadlineExceeded {
		log.Debug("Deregistering service from the directory timed out")
	} else if err != nil {
		log.WithField("error", err).Debug("Error while deregistering service from the directory")
	}
}

func ManageRegistration(
	ctx context.Context,
	port uint,
	protocol api.ServiceEndpoint_Protocol,
	serviceType api.ServiceType,
	options RegistrationOptions,
) error {
	if !options.DirectoryEndpoint.IsValid() {
		return nil
	}

	host := options.DirectoryRegistrationHost
	if host == "" {
		ip, err := getOutboundIP()
		if err != nil {
			return fmt.Errorf(
				"Unable to self discover the host ip, consider providing an explicit directory host [%w]",
				err,
			)
		}
		host = ip.String()
		options.DirectoryRegistrationProperties[endpoint.RegistrationSourcePropertyName] = "Self-Implicit"
	} else {
		options.DirectoryRegistrationProperties[endpoint.RegistrationSourcePropertyName] = "Self-Command_Line"
	}
	options.DirectoryRegistrationProperties[endpoint.VersionPropertyName] = version.Version

	log := log.WithFields(logrus.Fields{
		"protocol": protocol,
		"host":     host,
		"port":     port,
		"type":     serviceType,
		"endpoint": options.DirectoryEndpoint,
	})

	directoryClient, err := CreateClient(ctx, options.DirectoryEndpoint, options.DirectoryAuthToken)
	if err != nil {
		return err
	}

	log.Debug("Registering service to the directory...")

	serviceID, serviceSecret, err := directoryClient.Register(&api.RegisterRequest{
		Endpoint: &api.ServiceEndpoint{
			Protocol: protocol,
			Host:     host,
			Port:     uint32(port),
		},
		Details: &api.ServiceDetails{
			Type:       serviceType,
			Properties: options.DirectoryRegistrationProperties,
		},
		Permanent: false,
	})
	if err != nil {
		return err
	}

	defer deregisterService(log, options.DirectoryEndpoint, options.DirectoryAuthToken, serviceID, serviceSecret)

	// Awaiting context to be done
	<-ctx.Done()

	if ctx.Err() == context.Canceled {
		return nil
	}
	return ctx.Err()
}
