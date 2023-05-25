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
	"fmt"
	"net"

	"github.com/cogment/cogment/services/directory/grpcservers"
	"github.com/cogment/cogment/services/utils"
)

type Options struct {
	Port                uint
	GrpcReflection      bool
	RegistrationLag     uint
	PersistenceFilename string
}

var DefaultOptions = Options{
	Port:                9005,
	GrpcReflection:      false,
	RegistrationLag:     0,
	PersistenceFilename: ".cogment-directory-data",
}

func Run(options Options) error {
	server := utils.NewGrpcServer(options.GrpcReflection)
	dirServer, err := grpcservers.RegisterDirectoryServer(server, options.RegistrationLag, options.PersistenceFilename)
	if err != nil {
		return err
	}

	stopChecks, err := dirServer.PeriodicHealthCheck()
	if err != nil {
		return err
	}
	defer stopChecks()

	portString := fmt.Sprintf(":%d", options.Port)
	listener, err := net.Listen("tcp", portString)
	if err != nil {
		return fmt.Errorf("Unable to open TCP port [%d]: %v", options.Port, err)
	}

	log.WithField("port", options.Port).Info("Listening")
	err = server.Serve(listener)
	log.Info("Closing")

	saveErr := dirServer.SaveDatabase()
	if saveErr != nil {
		log.Warn("Failed to save database on close - ", saveErr)
	}

	return err
}
