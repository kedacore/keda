// Copyright 2018 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	catalog "github.com/operator-framework/operator-sdk/commands/operator-sdk/cmd/olm-catalog"

	"github.com/spf13/cobra"
)

func NewOLMCatalogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "olm-catalog <olm-catalog-command>",
		Short: "Invokes a olm-catalog command",
		Long: `The operator-sdk olm-catalog command invokes a command to perform
Catalog related actions.`,
	}
	cmd.AddCommand(catalog.NewGenCSVCmd())
	return cmd
}
