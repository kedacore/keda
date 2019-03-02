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

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"
)

// Tag applied to images that were pulled by digest. This denotes that the
// image was (probably) never tagged with this, but lets us avoid applying the
// ":latest" tag which might be misleading.
const iWasADigestTag = "i-was-a-digest"

func init() { Root.AddCommand(NewCmdPull()) }

// NewCmdPull creates a new cobra.Command for the pull subcommand.
func NewCmdPull() *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Pull a remote image by reference and store its contents in a tarball",
		Args:  cobra.ExactArgs(2),
		Run:   pull,
	}
}

func pull(_ *cobra.Command, args []string) {
	src, dst := args[0], args[1]

	ref, err := name.ParseReference(src, name.WeakValidation)
	if err != nil {
		log.Fatalf("parsing tag %q: %v", src, err)
	}
	log.Printf("Pulling %v", ref)

	i, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		log.Fatalf("reading image %q: %v", ref, err)
	}

	// WriteToFile wants a tag to write to the tarball, but we might have
	// been given a digest.
	// If the original ref was a tag, use that. Otherwise, if it was a
	// digest, tag the image with :i-was-a-digest instead.
	tag, ok := ref.(name.Tag)
	if !ok {
		d, ok := ref.(name.Digest)
		if !ok {
			log.Fatal("ref wasn't a tag or digest")
		}
		s := fmt.Sprintf("%s:%s", d.Repository.Name(), iWasADigestTag)
		tag, err = name.NewTag(s, name.WeakValidation)
		if err != nil {
			log.Fatalf("parsing digest as tag (%s): %v", s, err)
		}
	}

	if err := tarball.WriteToFile(dst, tag, i); err != nil {
		log.Fatalf("writing image %q: %v", dst, err)
	}
}
