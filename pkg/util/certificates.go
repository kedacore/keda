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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const customCAPath = "/tmp/custom/ca"

var logger = logf.Log.WithName("certificates")

var (
	rootCAs     *x509.CertPool
	rootCAsLock sync.Mutex
)

func getRootCAs() *x509.CertPool {
	rootCAsLock.Lock()
	defer rootCAsLock.Unlock()

	if rootCAs != nil {
		return rootCAs
	}

	var err error
	rootCAs, err = x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
		if err != nil {
			logger.Error(err, "failed to load system cert pool, using new cert pool instead")
		} else {
			logger.V(1).Info("system cert pool not available, using new cert pool instead")
		}
	}
	if _, err := os.Stat(customCAPath); errors.Is(err, fs.ErrNotExist) {
		logger.V(1).Info(fmt.Sprintf("the path %s doesn't exist, skipping custom CA registrations", customCAPath))
		return rootCAs
	}

	files, err := os.ReadDir(customCAPath)
	if err != nil {
		logger.Error(err, fmt.Sprintf("unable to read %s", customCAPath))
		return rootCAs
	}

	for _, file := range files {
		filename := file.Name()
		if file.IsDir() || strings.HasPrefix(filename, "..") {
			logger.V(1).Info(fmt.Sprintf("%s isn't a valid certificate", filename))
			continue // Skip directories and special files
		}

		filePath := filepath.Join(customCAPath, filename)
		certs, err := os.ReadFile(filePath)
		if err != nil {
			logger.Error(err, fmt.Sprintf("error reading %q", filename))
			continue
		}

		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			logger.Error(fmt.Errorf("no certs appended"), "filename", filename)
			continue
		}
		logger.V(1).Info(fmt.Sprintf("the certificate %s has been added to the pool", filename))
	}

	return rootCAs
}
