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

package directory

import (
	"fmt"
	"net"

	"github.com/cogment/cogment/services/directory/grpcservers"
	"github.com/cogment/cogment/services/utils"
)

type Options struct {
	Port           uint
	GrpcReflection bool
}

var DefaultOptions = Options{
	Port:           9005,
	GrpcReflection: false,
}

func Run(options Options) error {
	portString := fmt.Sprintf(":%d", options.Port)
	listener, err := net.Listen("tcp", portString)
	if err != nil {
		return fmt.Errorf("Unable to open TCP port [%d]: %v", options.Port, err)
	}
	server := utils.NewGrpcServer(options.GrpcReflection)
	err = grpcservers.RegisterDirectoryServer(server)
	if err != nil {
		return err
	}

	log.WithField("port", options.Port).Info("Listening")
	return server.Serve(listener)
}
