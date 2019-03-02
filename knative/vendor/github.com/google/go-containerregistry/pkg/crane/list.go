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
	"fmt"
	"log"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/cobra"
)

func init() { Root.AddCommand(NewCmdList()) }

// NewCmdList creates a new cobra.Command for the ls subcommand.
func NewCmdList() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List the tags in a repo",
		Args:  cobra.ExactArgs(1),
		Run:   ls,
	}
}

func ls(_ *cobra.Command, args []string) {
	r := args[0]
	repo, err := name.NewRepository(r, name.WeakValidation)
	if err != nil {
		log.Fatalf("parsing repo %q: %v", r, err)
	}

	auth, err := authn.DefaultKeychain.Resolve(repo.Registry)
	if err != nil {
		log.Fatalf("getting creds for %q: %v", repo, err)
	}

	tags, err := remote.List(repo, auth, http.DefaultTransport)
	if err != nil {
		log.Fatalf("reading tags for %q: %v", repo, err)
	}

	for _, tag := range tags {
		fmt.Println(tag)
	}
}
