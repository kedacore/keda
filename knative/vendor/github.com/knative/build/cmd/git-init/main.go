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
package main

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/knative/pkg/logging"
	"go.uber.org/zap"
)

var (
	url      = flag.String("url", "", "The url of the Git repository to initialize.")
	revision = flag.String("revision", "", "The Git revision to make the repository HEAD")
	path     = flag.String("path", "", "Path of directory under which git repository will be copied")
)

func run(logger *zap.SugaredLogger, cmd string, args ...string) {
	c := exec.Command(cmd, args...)
	var output bytes.Buffer
	c.Stderr = &output
	c.Stdout = &output
	if err := c.Run(); err != nil {
		logger.Errorf("Error running %v %v: %v\n%v", cmd, args, err, output.String())
	}
}

func runOrFail(logger *zap.SugaredLogger, cmd string, args ...string) {
	c := exec.Command(cmd, args...)
	var output bytes.Buffer
	c.Stderr = &output
	c.Stdout = &output

	if err := c.Run(); err != nil {
		logger.Fatalf("Unexpected error running %v %v: %v\n%v", cmd, args, err, output.String())
	}
}

func main() {
	flag.Parse()
	logger, _ := logging.NewLogger("", "git-init")
	defer logger.Sync()

	// HACK HACK HACK
	// Git seems to ignore $HOME/.ssh and look in /root/.ssh for unknown reasons.
	// As a workaround, symlink /root/.ssh to where we expect the $HOME to land.
	// This means SSH auth only works for our built-in git support, and not
	// custom steps.
	err := os.Symlink("/builder/home/.ssh", "/root/.ssh")
	if err != nil {
		logger.Fatalf("Unexpected error creating symlink: %v", err)
	}

	dir, err := os.Getwd()
	if err != nil {
		logger.Errorf("Failed to get current dir", err)
	}

	if *path != "" {
		runOrFail(logger, "git", "init", *path)
		path := filepath.Join(dir, *path)
		if err := os.Chdir(path); err != nil {
			logger.Fatalf("Failed to change directory with path %s; err %v", path, err)
		}
		// update dir variable with new path
		dir = path
	} else {
		run(logger, "git", "init")
	}

	run(logger, "git", "remote", "add", "origin", *url)
	runOrFail(logger, "git", "fetch", "--depth=1", "--recurse-submodules=yes", "origin", *revision)
	runOrFail(logger, "git", "reset", "--hard", "FETCH_HEAD")

	logger.Infof("Successfully cloned %q @ %q in path %q", *url, *revision, dir)
}
