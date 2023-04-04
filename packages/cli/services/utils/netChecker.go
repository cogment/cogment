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

package utils

import (
	"fmt"
	"net"
)

// Done this way to match the C++ version
type NetChecker struct {
}

func NewNetChecker() *NetChecker {
	nc := new(NetChecker)
	return nc
}

func (nc *NetChecker) TCPPort(port uint16) error {
	portString := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", portString)
	if err != nil {
		return err
	}
	listener.Close()

	return nil
}

func (nc *NetChecker) IsTCPPortUsed(port uint16) bool {
	err := nc.TCPPort(port)
	return (err != nil)
}
