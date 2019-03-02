/*
Copyright 2017 Google Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/knative/build/pkg/logs"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var namespace = flag.String("n", "default", "The namespace scope for this CLI request")

func main() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		log.Fatalf("Usage: %s [-n NAMESPACE] BUILD-NAME\n", os.Args[0])
	}

	buildName := flag.Args()[0]
	ctx := context.Background()

	if err := logs.Tail(ctx, os.Stdout, buildName, *namespace); err != nil {
		log.Fatalln(err)
	}
}
