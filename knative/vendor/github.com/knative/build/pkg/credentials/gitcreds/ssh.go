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
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/knative/build/pkg/credentials"
)

const sshKnownHosts = "known_hosts"

// As the flag is read, this status is populated.
// sshGitConfig implements flag.Value
type sshGitConfig struct {
	entries map[string]sshEntry
	// The order we see things, for iterating over the above.
	order []string
}

func (dc *sshGitConfig) String() string {
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

func (dc *sshGitConfig) Set(value string) error {
	parts := strings.Split(value, "=")
	if len(parts) != 2 {
		return fmt.Errorf("Expect entries of the form secret=url, got: %v", value)
	}
	secret := parts[0]
	url := parts[1]

	if _, ok := dc.entries[url]; ok {
		return fmt.Errorf("Multiple entries for url: %v", url)
	}

	e, err := newSshEntry(url, secret)
	if err != nil {
		return err
	}
	dc.entries[url] = *e
	dc.order = append(dc.order, url)
	return nil
}

func (dc *sshGitConfig) Write() error {
	sshDir := filepath.Join(os.Getenv("HOME"), ".ssh")
	if err := os.MkdirAll(sshDir, os.ModePerm); err != nil {
		return err
	}

	// Walk each of the entries and for each do three things:
	//  1. Write out: ~/.ssh/id_{secret} with the secret key
	//  2. Compute its part of "~/.ssh/config"
	//  3. Compute its part of "~/.ssh/known_hosts"
	var configEntries []string
	var defaultPort = "22"
	var knownHosts []string
	for _, k := range dc.order {
		var host, port string
		var err error
		if host, port, err = net.SplitHostPort(k); err != nil {
			host = k
			port = defaultPort
		}
		v := dc.entries[k]
		if err := v.Write(sshDir); err != nil {
			return err
		}
		configEntries = append(configEntries, fmt.Sprintf(`Host %s
    HostName %s
    IdentityFile %s
    Port %s
`, host, host, v.path(sshDir), port))

		knownHosts = append(knownHosts, v.knownHosts)
	}
	configPath := filepath.Join(sshDir, "config")
	configContent := strings.Join(configEntries, "")
	if err := ioutil.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		return err
	}
	knownHostsPath := filepath.Join(sshDir, "known_hosts")
	knownHostsContent := strings.Join(knownHosts, "\n")
	return ioutil.WriteFile(knownHostsPath, []byte(knownHostsContent), 0600)
}

type sshEntry struct {
	secret     string
	privateKey string
	knownHosts string
}

func (be *sshEntry) path(sshDir string) string {
	return filepath.Join(sshDir, "id_"+be.secret)
}

func sshKeyScan(domain string) ([]byte, error) {
	c := exec.Command("ssh-keyscan", domain)
	var output bytes.Buffer
	c.Stdout = &output
	c.Stderr = &output
	if err := c.Run(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func (be *sshEntry) Write(sshDir string) error {
	return ioutil.WriteFile(be.path(sshDir), []byte(be.privateKey), 0600)
}

func newSshEntry(u, secret string) (*sshEntry, error) {
	secretPath := credentials.VolumeName(secret)

	pk, err := ioutil.ReadFile(filepath.Join(secretPath, corev1.SSHAuthPrivateKey))
	if err != nil {
		return nil, err
	}
	privateKey := string(pk)

	kh, err := ioutil.ReadFile(filepath.Join(secretPath, sshKnownHosts))
	if err != nil {
		kh, err = sshKeyScan(u)
		if err != nil {
			return nil, err
		}
	}
	knownHosts := string(kh)

	return &sshEntry{
		secret:     secret,
		privateKey: privateKey,
		knownHosts: knownHosts,
	}, nil
}
