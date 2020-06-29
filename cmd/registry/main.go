package registry

import (
	"github.com/spf13/cobra"
)

func NewRegistryCommand(verbose bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Interact with Docker registries configuration",
		Hidden: true,
	}

	cmd.AddCommand(NewRegistryConfigureCommand(verbose))
	cmd.AddCommand(NewRegistryListCommand(verbose))
	cmd.AddCommand(NewRegistryDeleteCommand(verbose))

	return cmd
}
