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
