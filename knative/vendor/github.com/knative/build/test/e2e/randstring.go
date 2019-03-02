/*
Copyright 2018 Knative Authors LLC

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

package e2e

import (
	"math/rand"
	"strings"
	"sync"
	"time"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyz"
	randSuffixLen = 8
)

// r is used by AppendRandomString to generate a random string. It is seeded with the time
// at import so the strings will be different between test runs.
var (
	r        *rand.Rand
	rndMutex *sync.Mutex
)

// once is used to initialize r
var once sync.Once

func initSeed() {
	seed := time.Now().UTC().UnixNano()
	r = rand.New(rand.NewSource(seed))
	rndMutex = &sync.Mutex{}
}

// AppendRandomString will generate a random string that begins with prefix. This is useful
// if you want to make sure that your tests can run at the same time against the same
// environment without conflicting. This method will seed rand with the current time when
// called for the first time.
// TODO(https://github.com/knative/pkg/issues/226): Use a shared version from knative/pkg.
func AppendRandomString(prefix string) string {
	once.Do(initSeed)
	suffix := make([]byte, randSuffixLen)
	rndMutex.Lock()
	for i := range suffix {
		suffix[i] = letterBytes[r.Intn(len(letterBytes))]
	}
	rndMutex.Unlock()
	return strings.Join([]string{prefix, string(suffix)}, "-")
}
