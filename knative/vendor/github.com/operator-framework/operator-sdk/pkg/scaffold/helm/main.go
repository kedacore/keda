// Copyright 2019 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helm

import (
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

// Main - main source file for helm operator
type Main struct {
	input.Input
}

func (m *Main) GetInput() (input.Input, error) {
	if m.Path == "" {
		m.Path = filepath.Join("cmd", "manager", "main.go")
	}
	m.TemplateBody = mainTmpl
	return m.Input, nil
}

const mainTmpl = `package main

import (
	"os"

	hoflags "github.com/operator-framework/operator-sdk/pkg/helm/flags"
	"github.com/operator-framework/operator-sdk/pkg/helm"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"

	"github.com/spf13/pflag"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func main() {
	hflags := hoflags.AddTo(pflag.CommandLine)
	pflag.Parse()
	logf.SetLogger(zap.Logger())

	if err := helm.Run(hflags); err != nil {
		logf.Log.WithName("cmd").Error(err, "")
		os.Exit(1)
	}
}
`
