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

package gitcreds

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/knative/build/pkg/credentials"
)

// As the flag is read, this status is populated.
// basicGitConfig implements flag.Value
type basicGitConfig struct {
	entries map[string]basicEntry
	// The order we see things, for iterating over the above.
	order []string
}

func (dc *basicGitConfig) String() string {
	if dc == nil {
		// According to flag.Value this can happen.
		return ""
	}
	var urls []string
	for _, k := range dc.order {
		v := dc.entries[k]
		urls = append(urls, fmt.Sprintf("%s=%s", v.secret, k))
	}
	return strings.Join(urls, ",")
}

func (dc *basicGitConfig) Set(value string) error {
	parts := strings.Split(value, "=")
	if len(parts) != 2 {
		return fmt.Errorf("Expect entries of the form secret=url, got: %v", value)
	}
	secret := parts[0]
	url := parts[1]

	if _, ok := dc.entries[url]; ok {
		return fmt.Errorf("Multiple entries for url: %v", url)
	}

	e, err := newBasicEntry(url, secret)
	if err != nil {
		return err
	}
	dc.entries[url] = *e
	dc.order = append(dc.order, url)
	return nil
}

func (dc *basicGitConfig) Write() error {
	gitConfigPath := filepath.Join(os.Getenv("HOME"), ".gitconfig")
	gitConfigs := []string{
		"[credential]\n	helper = store\n",
	}
	for _, k := range dc.order {
		v := dc.entries[k]
		gitConfigs = append(gitConfigs, v.configBlurb(k))
	}
	gitConfigContent := strings.Join(gitConfigs, "")
	if err := ioutil.WriteFile(gitConfigPath, []byte(gitConfigContent), 0600); err != nil {
		return err
	}

	gitCredentialsPath := filepath.Join(os.Getenv("HOME"), ".git-credentials")
	var gitCredentials []string
	for _, k := range dc.order {
		v := dc.entries[k]
		gitCredentials = append(gitCredentials, v.authURL.String())
	}
	gitCredentials = append(gitCredentials, "") // Get a trailing newline
	gitCredentialsContent := strings.Join(gitCredentials, "\n")
	return ioutil.WriteFile(gitCredentialsPath, []byte(gitCredentialsContent), 0600)
}

type basicEntry struct {
	secret   string
	username string
	password string
	// Has the form: https://user:pass@url.com
	authURL *url.URL
}

func (be *basicEntry) configBlurb(u string) string {
	return fmt.Sprintf("[credential %q]\n	username = %s\n", u, be.username)
}

func newBasicEntry(u, secret string) (*basicEntry, error) {
	secretPath := credentials.VolumeName(secret)

	ub, err := ioutil.ReadFile(filepath.Join(secretPath, corev1.BasicAuthUsernameKey))
	if err != nil {
		return nil, err
	}
	username := string(ub)

	pb, err := ioutil.ReadFile(filepath.Join(secretPath, corev1.BasicAuthPasswordKey))
	if err != nil {
		return nil, err
	}
	password := string(pb)

	pu, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	pu.User = url.UserPassword(username, password)

	return &basicEntry{
		secret:   secret,
		username: username,
		password: password,
		authURL:  pu,
	}, nil
}
