/*
Copyright 2018 The Knative Authors

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

package utils

import (
	"bufio"
	"io"
	"os"
	"strings"
	"sync"
)

const (
	resolverFileName  = "/etc/resolv.conf"
	defaultDomainName = "cluster.local"
)

var (
	domainName string
	once       sync.Once
)

// GetClusterDomainName returns cluster's domain name or an error
// Closes issue: https://github.com/knative/eventing/issues/714
func GetClusterDomainName() string {
	once.Do(func() {
		f, err := os.Open(resolverFileName)
		if err == nil {
			defer f.Close()
			domainName = getClusterDomainName(f)

		} else {
			domainName = defaultDomainName
		}
	})

	return domainName
}

func getClusterDomainName(r io.Reader) string {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		elements := strings.Split(scanner.Text(), " ")
		if elements[0] != "search" {
			continue
		}
		for i := 1; i < len(elements)-1; i++ {
			if strings.HasPrefix(elements[i], "svc.") {
				return elements[i][4:]
			}
		}
	}
	// For all abnormal cases return default domain name
	return defaultDomainName
}
