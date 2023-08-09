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
	"strconv"
	"strings"
)

// Expects a string of the form "<ip address>:<port>"
func ExtractPort(address string) (uint, error) {
	portIndex := strings.LastIndex(address, ":")
	if portIndex == -1 {
		return 0, fmt.Errorf("Invalid address format [%v]", address)
	}

	portStr := address[portIndex+1:]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("Failed to convert port value [%v] - %w", address, err)
	}

	if port < 0 || port > 65535 {
		return 0, fmt.Errorf("Invalid port number [%v]", address)
	}

	return uint(port), nil
}
