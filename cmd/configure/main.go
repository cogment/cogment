package configure

import (
	"github.com/spf13/cobra"
)

func NewConfigureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "configure",
		Short:  "Configure Cogment CLI options",
		Hidden: true,
	}

	cmd.AddCommand(NewRemoteCmd())

	return cmd
}
