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

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/cogment/cogment/deployment"
	"log"
)

//var imagesOnly, manifestOnly bool

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:    "build",
	Short:  "Build your images for the deployment",
	Hidden: true,
	//	Long: `A longer description that spans multiple lines and likely contains examples
	//and usage of using your command. For example:
	//
	//Cobra is a CLI library for Go that empowers applications.
	//This application is a tool to generate the needed files
	//to quickly create a Cobra application.`,

	//PreRun: func(cmd *cobra.Command, args []string) {
	//	if !manifestOnly && !imagesOnly {
	//		manifestOnly = true
	//		imagesOnly = true
	//	}
	//},

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Verbose in run is %t", Verbose)
		runBuildCmd(args)
	},
}

func runBuildCmd(services []string) {
	manifest, err := deployment.CreateManifestFromCompose("docker-compose.yaml", services)
	if err != nil {
		log.Fatalln(err)
	}

	if len(manifest.Services) < 1 {
		log.Fatalln("We found no service to deploy")
	}

	fmt.Printf("We found %d services to deploy\n", len(manifest.Services))

	deployment.BuildImages(manifest)

}

func init() {
	rootCmd.AddCommand(buildCmd)

	//buildCmd.Flags().BoolVar(&imagesOnly, "images", false, "This will only build the docker images")
	//buildCmd.Flags().BoolVar(&manifestOnly, "manifest", false, "This will only build the manifest")
}