/*
Copyright 2023 The KEDA Authors

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

package util

import (
	"crypto/x509"
	"fmt"
	"os"
	"path"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const customCAPath = "/custom-cas"

var logger = logf.Log.WithName("certificates")

var rootCAs *x509.CertPool

func init() {
	certPool, _ := x509.SystemCertPool()
	if certPool == nil {
		certPool = x509.NewCertPool()
	}

	files, err := os.ReadDir(customCAPath)
	if err != nil {
		logger.Error(err, fmt.Sprintf("unable to read %s", customCAPath))
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		certs, err := os.ReadFile(path.Join(customCAPath, file.Name()))
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to append %q to certPool", file.Name()))
		}

		if ok := certPool.AppendCertsFromPEM(certs); !ok {
			logger.Error(fmt.Errorf("no certs appended"), fmt.Sprintf("the certificate %s hasn't been added to the pool", file.Name()))
		}
		logger.V(1).Info(fmt.Sprintf("the certificate %s has been added to the pool", file.Name()))
	}

	rootCAs = certPool
}

func getRootCAs() *x509.CertPool {
	return rootCAs
}
