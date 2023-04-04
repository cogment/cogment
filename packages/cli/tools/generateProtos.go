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

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	apiDir, apiDirFound := os.LookupEnv("COGMENT_API_DIR")
	if !apiDirFound {
		fmt.Printf("COGMENT_API_DIR environment variable requires to be set to the path to the api\n")
		os.Exit(-1)
	}

	protocExec, protocFound := os.LookupEnv("PROTOC_EXEC")
	if !protocFound {
		fmt.Printf("PROTOC_EXEC environment variable is not defined, using default value \"protoc\"\n")
		protocExec = "protoc"
	}

	packageDir := "grpcapi"

	fmt.Printf("Generating go code from the cogment API from %q using %q...\n", apiDir, protocExec)

	err := os.MkdirAll(packageDir, 0755)
	if err != nil {
		fmt.Printf("Error while creating package directory: %s\n", packageDir)
		os.Exit(-1)
	}

	protoFiles := []string{
		"cogment/api/agent.proto",
		"cogment/api/common.proto",
		"cogment/api/datalog.proto",
		"cogment/api/directory.proto",
		"cogment/api/environment.proto",
		"cogment/api/health.proto",
		"cogment/api/hooks.proto",
		"cogment/api/model_registry.proto",
		"cogment/api/orchestrator.proto",
		"cogment/api/trial_datastore.proto",
	}
	protocArgs := []string{
		fmt.Sprintf("--proto_path=%s", apiDir),
		fmt.Sprintf("--go_out=%s", packageDir),
		fmt.Sprintf("--go-grpc_out=%s", packageDir),
		"--go_opt=paths=source_relative",
		"--go-grpc_opt=paths=source_relative",
	}

	if protobufIncludeDir, protobufIncludeDirFound := os.LookupEnv("PROTOBUF_INCLUDE_DIR"); protobufIncludeDirFound {
		protocArgs = append(protocArgs, fmt.Sprintf("--proto_path=%s", protobufIncludeDir))
	}

	protocArgs = append(protocArgs, protoFiles...)

	protocCmd := exec.Command(protocExec, protocArgs...)

	protocCmd.Stdout = os.Stdout
	protocCmd.Stderr = os.Stderr

	err = protocCmd.Run()
	if err != nil {
		fmt.Printf("Error while running protoc: %s\n", err)
		os.Exit(-1)
	}

	os.Exit(0)
}
