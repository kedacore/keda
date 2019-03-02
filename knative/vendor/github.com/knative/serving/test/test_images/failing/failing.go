/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"log"
	"os"
	"time"
)

func main() {
	log.Println("Started...")

	// Sleep for 10 seconds to force a race condition, where this
	// container becomes ready if no readinessProbe is set.
	time.Sleep(10 * time.Second)

	log.Println("Crashed...")
	os.Exit(5)
}
