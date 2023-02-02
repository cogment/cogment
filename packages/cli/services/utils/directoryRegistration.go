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

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/cogment/cogment/clients/directory"
	"github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/sirupsen/logrus"
)

// A DirectoryRegistrationOptions holds the options for a service to be registered with one or more directories
//
// This struct is designed to be used as a "mixin" in services options.
type DirectoryRegistrationOptions struct {
	DirectoryEndpoint               string
	DirectoryAuthToken              string
	DirectoryRegistrationHost       string
	DirectoryRegistrationProperties map[string]string
}

var DefaultDirectoryRegistrationOptions = DirectoryRegistrationOptions{
	DirectoryEndpoint:               "",
	DirectoryAuthToken:              "",
	DirectoryRegistrationHost:       "",
	DirectoryRegistrationProperties: map[string]string{},
}

func getOutboundIP() net.IP {
	host, _ := os.Hostname()
	hostIPs, _ := net.LookupIP(host)
	for _, hostIP := range hostIPs {
		if hostIP.IsLoopback() {
			continue
		}
		if hostIPv4 := hostIP.To4(); hostIPv4 != nil {
			return hostIPv4
		}
	}
	return nil
}

const serviceDeregisterTimeout = time.Second * 10

func deregisterService(
	log *logrus.Entry,
	directoryEndpoint string,
	directoryAuthToken string,
	serviceID uint64,
	serviceSecret string,
) {
	ctx, cancel := context.WithTimeout(context.Background(), serviceDeregisterTimeout)
	defer cancel()
	directoryClient, err := directory.CreateClient(ctx, directoryEndpoint, directoryAuthToken)
	if err != nil {
		log.WithField("failure", err).Debug("Error while deregistering service from the directory")
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
		log.WithField("failure", err).Debug("Error while deregistering service from the directory")
	}
}

func ManageDirectoryRegistration(
	ctx context.Context,
	port uint,
	protocol api.ServiceEndpoint_Protocol,
	serviceType api.ServiceType,
	options DirectoryRegistrationOptions,
) error {
	if len(options.DirectoryEndpoint) == 0 {
		return nil
	}

	host := options.DirectoryRegistrationHost
	if host == "" {
		host = getOutboundIP().String()
	}

	var log = logrus.WithFields(logrus.Fields{
		"component": "directory-registration",
		"protocol":  protocol,
		"host":      host,
		"port":      port,
		"type":      serviceType,
	})

	directoryClient, err := directory.CreateClient(ctx, options.DirectoryEndpoint, options.DirectoryAuthToken)
	if err != nil {
		return err
	}

	dirlog := log.WithFields(logrus.Fields{
		"endpoint": options.DirectoryEndpoint,
	})
	dirlog.Debug("Registering service to the directory...")

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

	defer deregisterService(dirlog, options.DirectoryEndpoint, options.DirectoryAuthToken, serviceID, serviceSecret)

	// Awaiting context to be done
	<-ctx.Done()

	if ctx.Err() == context.Canceled {
		return nil
	}
	return ctx.Err()
}
