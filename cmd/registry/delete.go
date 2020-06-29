package registry

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.com/cogment/cogment/deployment"
	"log"
	"net/http"
)

func runDeleteRegistryCmd(client *resty.Client, registryId string) error {
	resp, err := client.R().Delete("/docker-registries/" + registryId)

	if err != nil {
		log.Fatalf("%v", err)
	}

	if http.StatusNotFound == resp.StatusCode() {
		return fmt.Errorf("%s", "Registry ID not found")
	}

	return nil
}

func NewRegistryDeleteCommand(verbose bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete configured Docker Registry",
		Args:  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			client, err := deployment.PlatformClient(verbose)
			if err != nil {
				log.Fatal(err)
			}

			err = runDeleteRegistryCmd(client, args[0])

			if err != nil {
				log.Fatalf("%v", err)
			}

			fmt.Println("Registry deleted")
		},
	}

	return cmd
}
