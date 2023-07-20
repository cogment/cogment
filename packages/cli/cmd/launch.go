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

package cmd

import (
	"github.com/cogment/cogment/launcher"
	"github.com/spf13/cobra"
)

var launchCmd = &cobra.Command{
	Use:          "launch [-q[q[q]]] <filename> [args...]",
	Short:        "Launches and manages processes for local deployments",
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		quietLevel, err := cmd.Flags().GetCount("quiet")
		if err != nil {
			return err
		}

		return launcher.Launch(args, quietLevel)
	},
}

func init() {
	launchCmd.Flags().CountP("quiet", "q", "")
}
