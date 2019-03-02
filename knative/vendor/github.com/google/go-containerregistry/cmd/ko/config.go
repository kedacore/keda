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
	"log"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

var (
	defaultBaseImage   name.Reference
	baseImageOverrides map[string]name.Reference
)

func getBaseImage(s string) (v1.Image, error) {
	ref, ok := baseImageOverrides[s]
	if !ok {
		ref = defaultBaseImage
	}
	log.Printf("Using base %s for %s", ref, s)
	return remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
}

func getCreationTime() (*v1.Time, error) {
	epoch := os.Getenv("SOURCE_DATE_EPOCH")
	if epoch == "" {
		return nil, nil
	}

	seconds, err := strconv.ParseInt(epoch, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("the environment variable SOURCE_DATE_EPOCH is invalid. It's must be a number of seconds since January 1st 1970, 00:00 UTC, got %v", err)
	}
	return &v1.Time{time.Unix(seconds, 0)}, nil
}

func init() {
	// If omitted, use this base image.
	viper.SetDefault("defaultBaseImage", "gcr.io/distroless/static:latest")
	viper.SetConfigName(".ko") // .yaml is implicit

	if override := os.Getenv("KO_CONFIG_PATH"); override != "" {
		viper.AddConfigPath(override)
	}

	viper.AddConfigPath("./")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatalf("error reading config file: %v", err)
		}
	}

	ref := viper.GetString("defaultBaseImage")
	dbi, err := name.ParseReference(ref, name.WeakValidation)
	if err != nil {
		log.Fatalf("'defaultBaseImage': error parsing %q as image reference: %v", ref, err)
	}
	defaultBaseImage = dbi

	baseImageOverrides = make(map[string]name.Reference)
	overrides := viper.GetStringMapString("baseImageOverrides")
	for k, v := range overrides {
		bi, err := name.ParseReference(v, name.WeakValidation)
		if err != nil {
			log.Fatalf("'baseImageOverrides': error parsing %q as image reference: %v", v, err)
		}
		baseImageOverrides[k] = bi
	}
}
