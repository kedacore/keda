//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Tomasz Mielech
//

package driver

import (
	"context"
)

type FoxxCreateOptions struct {
	Mount string
}

type FoxxDeleteOptions struct {
	Mount    string
	Teardown bool
}

type ClientFoxx interface {
	Foxx() FoxxService
}

type FoxxService interface {
	// InstallFoxxService installs a new service at a given mount path.
	InstallFoxxService(ctx context.Context, zipFile string, options FoxxCreateOptions) error
	// UninstallFoxxService uninstalls service at a given mount path.
	UninstallFoxxService(ctx context.Context, options FoxxDeleteOptions) error
}
