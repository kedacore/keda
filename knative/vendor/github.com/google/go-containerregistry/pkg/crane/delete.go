// Copyright 2018 Google LLC All Rights Reserved.
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

package crane

import (
	"log"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/cobra"
)

func init() { Root.AddCommand(NewCmdDelete()) }

// NewCmdDelete creates a new cobra.Command for the delete subcommand.
func NewCmdDelete() *cobra.Command {
	return &cobra.Command{
		Use:   "delete",
		Short: "Delete an image reference from its registry",
		Args:  cobra.ExactArgs(1),
		Run:   doDelete,
	}
}

func doDelete(_ *cobra.Command, args []string) {
	ref := args[0]
	r, err := name.ParseReference(ref, name.WeakValidation)
	if err != nil {
		log.Fatalf("parsing reference %q: %v", ref, err)
	}

	auth, err := authn.DefaultKeychain.Resolve(r.Context().Registry)
	if err != nil {
		log.Fatalf("getting creds for %q: %v", r, err)
	}

	if err := remote.Delete(r, auth, http.DefaultTransport); err != nil {
		log.Fatalf("deleting image %q: %v", r, err)
	}
}
