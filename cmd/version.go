package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/cogment/cogment/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of the Cogment CLI",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.CliVersion)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
