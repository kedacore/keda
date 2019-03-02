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

package main

import (
	"fmt"
	gb "go/build"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/ko/build"
	"github.com/google/go-containerregistry/pkg/ko/publish"
	"github.com/google/go-containerregistry/pkg/name"
)

func qualifyLocalImport(importpath, gopathsrc, pwd string) (string, error) {
	if !strings.HasPrefix(pwd, gopathsrc) {
		return "", fmt.Errorf("pwd (%q) must be on $GOPATH/src (%q) to support local imports", pwd, gopathsrc)
	}
	// Given $GOPATH/src and $PWD (which must be within $GOPATH/src), trim
	// off $GOPATH/src/ from $PWD and append local importpath to get the
	// fully-qualified importpath.
	return filepath.Join(strings.TrimPrefix(pwd, gopathsrc+string(filepath.Separator)), importpath), nil
}

func publishImages(importpaths []string, no *NameOptions, lo *LocalOptions, ta *TagsOptions) map[string]name.Reference {
	opt, err := gobuildOptions()
	if err != nil {
		log.Fatalf("error setting up builder options: %v", err)
	}
	b, err := build.NewGo(opt...)
	if err != nil {
		log.Fatalf("error creating go builder: %v", err)
	}
	imgs := make(map[string]name.Reference)
	for _, importpath := range importpaths {
		if gb.IsLocalImport(importpath) {
			// Qualify relative imports to their fully-qualified
			// import path, assuming $PWD is within $GOPATH/src.
			gopathsrc := filepath.Join(gb.Default.GOPATH, "src")
			pwd, err := os.Getwd()
			if err != nil {
				log.Fatalf("error getting current working directory: %v", err)
			}
			importpath, err = qualifyLocalImport(importpath, gopathsrc, pwd)
			if err != nil {
				log.Fatal(err)
			}
		}

		if !b.IsSupportedReference(importpath) {
			log.Fatalf("importpath %q is not supported", importpath)
		}

		img, err := b.Build(importpath)
		if err != nil {
			log.Fatalf("error building %q: %v", importpath, err)
		}
		var pub publish.Interface
		repoName := os.Getenv("KO_DOCKER_REPO")

		var namer publish.Namer
		if no.PreserveImportPaths {
			namer = preserveImportPath
		} else if no.BaseImportPaths {
			namer = baseImportPaths
		} else {
			namer = packageWithMD5
		}

		if lo.Local || repoName == publish.LocalDomain {
			pub = publish.NewDaemon(namer, ta.Tags)
		} else {
			if _, err := name.NewRepository(repoName, name.WeakValidation); err != nil {
				log.Fatalf("the environment variable KO_DOCKER_REPO must be set to a valid docker repository, got %v", err)
			}
			opts := []publish.Option{publish.WithAuthFromKeychain(authn.DefaultKeychain), publish.WithNamer(namer), publish.WithTags(ta.Tags)}
			pub, err = publish.NewDefault(repoName, opts...)
			if err != nil {
				log.Fatalf("error setting up default image publisher: %v", err)
			}
		}
		ref, err := pub.Publish(img, importpath)
		if err != nil {
			log.Fatalf("error publishing %s: %v", importpath, err)
		}
		imgs[importpath] = ref
	}
	return imgs
}
