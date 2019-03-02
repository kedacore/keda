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

package changeset

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	commitIDFile      = "HEAD"
	koDataPathEnvName = "KO_DATA_PATH"
)

var (
	commitIDRE = regexp.MustCompile(`^[a-f0-9]{40}$`)
)

// Get tries to fetch the first 7 digitals of GitHub commit ID from HEAD file in
// KO_DATA_PATH. If it fails, it returns the error it gets.
func Get() (string, error) {
	data, err := readFileFromKoData(commitIDFile)
	if err != nil {
		return "", err
	}
	commitID := strings.TrimSpace(string(data))
	if !commitIDRE.MatchString(commitID) {
		err := fmt.Errorf("%q is not a valid GitHub commit ID", commitID)
		return "", err
	}
	return string(commitID[0:7]), nil
}

// readFileFromKoData tries to read data as string from the file with given name
// under KO_DATA_PATH then returns the content as string. The file is expected
// to be wrapped into the container from /kodata by ko. If it fails, returns
// the error it gets.
func readFileFromKoData(filename string) ([]byte, error) {
	koDataPath := os.Getenv(koDataPathEnvName)
	if koDataPath == "" {
		err := fmt.Errorf("%q does not exist or is empty", koDataPathEnvName)
		return nil, err
	}
	fullFilename := filepath.Join(koDataPath, filename)
	return ioutil.ReadFile(fullFilename)
}
