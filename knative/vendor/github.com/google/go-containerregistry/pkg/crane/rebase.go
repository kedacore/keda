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
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/cobra"
)

func init() { Root.AddCommand(NewCmdRebase()) }

// NewCmdRebase creates a new cobra.Command for the rebase subcommand.
func NewCmdRebase() *cobra.Command {
	var orig, oldBase, newBase, rebased string
	rebaseCmd := &cobra.Command{
		Use:   "rebase",
		Short: "Rebase an image onto a new base image",
		Args:  cobra.NoArgs,
		Run: func(*cobra.Command, []string) {
			rebase(orig, oldBase, newBase, rebased)
		},
	}
	rebaseCmd.Flags().StringVarP(&orig, "original", "", "", "Original image to rebase")
	rebaseCmd.Flags().StringVarP(&oldBase, "old_base", "", "", "Old base image to remove")
	rebaseCmd.Flags().StringVarP(&newBase, "new_base", "", "", "New base image to insert")
	rebaseCmd.Flags().StringVarP(&rebased, "rebased", "", "", "Tag to apply to rebased image")

	rebaseCmd.MarkFlagRequired("original")
	rebaseCmd.MarkFlagRequired("old_base")
	rebaseCmd.MarkFlagRequired("new_base")
	rebaseCmd.MarkFlagRequired("rebased")
	return rebaseCmd
}

func rebase(orig, oldBase, newBase, rebased string) {
	if orig == "" || oldBase == "" || newBase == "" || rebased == "" {
		log.Fatalln("Must provide --original, --old_base, --new_base and --rebased")
	}

	origImg, _, err := getImage(orig)
	if err != nil {
		log.Fatalln(err)
	}

	oldBaseImg, _, err := getImage(oldBase)
	if err != nil {
		log.Fatalln(err)
	}

	newBaseImg, _, err := getImage(newBase)
	if err != nil {
		log.Fatalln(err)
	}

	rebasedTag, err := name.NewTag(rebased, name.WeakValidation)
	if err != nil {
		log.Fatalf("parsing tag %q: %v", rebased, err)
	}

	rebasedImg, err := mutate.Rebase(origImg, oldBaseImg, newBaseImg)
	if err != nil {
		log.Fatalf("rebasing: %v", err)
	}

	dig, err := rebasedImg.Digest()
	if err != nil {
		log.Fatalf("digesting rebased: %v", err)
	}

	auth, err := authn.DefaultKeychain.Resolve(rebasedTag.Context().Registry)
	if err != nil {
		log.Fatalf("getting creds for %q: %v", rebasedTag, err)
	}

	if err := remote.Write(rebasedTag, rebasedImg, auth, http.DefaultTransport); err != nil {
		log.Fatalf("writing image %q: %v", rebasedTag, err)
	}
	fmt.Print(dig.String())
}
