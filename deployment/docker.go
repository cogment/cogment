// Copyright 2020 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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

package deployment

import (
	"fmt"
	"log"
	"os/exec"
)

func BuildImages(m *DeploymentManifest) {

	for name, svc := range m.Services {
		log.Printf("Building docker image %s\n", svc.Image)

		cmd := exec.Command("docker-compose", "build", name)

		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("combined out:\n%s\n", string(out))
			log.Fatalf("cmd.Run() failed with %s\n", err)
		}

		fmt.Printf("combined out:\n%s\n", string(out))
	}
}

func PushImages(m *DeploymentManifest) {

	for _, svc := range m.Services {
		if svc.getPushImage() == false {
			continue
		}

		log.Printf("Pushing docker image %s\n", svc.Image)

		cmd := exec.Command("docker", "push", svc.Image)

		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("combined out:\n%s\n", string(out))
			log.Fatalf("cmd.Run() failed with %s\n", err)
		}

		fmt.Printf("combined out:\n%s\n", string(out))
	}
}
