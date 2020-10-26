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
	"bufio"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.com/cogment/cogment/deployment"
	"gitlab.com/cogment/cogment/helper"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type deployCmdContext struct {
	imagesPusher func(manifest *deployment.DeploymentManifest)
	force        bool
	client       *resty.Client
	stdin        io.Reader
}

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:    "deploy",
	Short:  "Deploy your project to the platform",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := deployment.PlatformClient(Verbose)
		if err != nil {
			log.Fatalln(err)
		}

		manifest, err := deployment.CreateManifestFromCompose("docker-compose.yaml", args)
		if err != nil {
			log.Fatalln(err)
		}

		if len(manifest.Services) < 1 {
			log.Fatalln("We found no service to deploy")
		}

		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			log.Fatal(err)
		}
		manifest.Force = force

		fmt.Printf("We found %d services to deploy\n", len(manifest.Services))

		ctx := deployCmdContext{
			imagesPusher: deployment.PushImages,
			client:       client,
			stdin:        os.Stdin,
		}

		err = runDeployCmd(manifest, &ctx)
		if err != nil {
			log.Println("DEPLOYMENT FAILED")
			log.Fatalln(err)
		}

		fmt.Println("You application has been submitted for deployment. Please run 'cogment inspect' to verify its status")
	},
}

func getAcceptDeployment(stdin io.Reader, appId string, manifest *deployment.DeploymentManifest) string {
	reader := bufio.NewReader(stdin)
	fmt.Println(helper.PrettyPrint(manifest))
	fmt.Printf("Do you want to deploy this manifest into application %s (y/N): ", appId)
	accept, _ := reader.ReadString('\n')
	accept = strings.TrimSpace(accept)
	return accept
}

func runDeployCmd(manifest *deployment.DeploymentManifest, ctx *deployCmdContext) error {

	appId := helper.CurrentConfig("app")
	if appId == "" {
		log.Fatal("No current application found, maybe try `cogment new`")
	}

	accept := getAcceptDeployment(ctx.stdin, appId, manifest)
	if accept != "y" {
		fmt.Println("Deployment aborted")
		os.Exit(0)
	}

	log.Println("Deploying your images")
	ctx.imagesPusher(manifest)
	//deployment.PushImages(manifest)

	log.Println("Pushing your project")
	resp, err := ctx.client.R().
		SetBody(manifest).
		Post(fmt.Sprintf("/applications/%s/deploy", appId))

	if err != nil {
		log.Fatalf("%v", err)
	}

	if http.StatusNotFound == resp.StatusCode() {
		return fmt.Errorf("%s", "Application not found")
	}

	if http.StatusCreated != resp.StatusCode() {
		return fmt.Errorf("%s", resp.Body())
	}

	return nil
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().Bool("force", false, "verbose output")

}
