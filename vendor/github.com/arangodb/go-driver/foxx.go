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
	"io/ioutil"
	"net/http"
	"strconv"
)

// InstallFoxxService installs a new service at a given mount path.
func (c *client) InstallFoxxService(ctx context.Context, zipFile string, options FoxxCreateOptions) error {

	req, err := c.conn.NewRequest("POST", "_api/foxx")
	if err != nil {
		return WithStack(err)
	}

	req.SetHeader("Content-Type", "application/zip")
	req.SetQuery("mount", options.Mount)

	bytes, err := ioutil.ReadFile(zipFile)
	if err != nil {
		return WithStack(err)
	}

	_, err = req.SetBody(bytes)
	if err != nil {
		return WithStack(err)
	}

	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}

	if err := resp.CheckStatus(http.StatusCreated); err != nil {
		return WithStack(err)
	}

	return nil
}

// UninstallFoxxService uninstalls service at a given mount path.
func (c *client) UninstallFoxxService(ctx context.Context, options FoxxDeleteOptions) error {
	req, err := c.conn.NewRequest("DELETE", "_api/foxx/service")
	if err != nil {
		return WithStack(err)
	}

	req.SetQuery("mount", options.Mount)
	req.SetQuery("teardown", strconv.FormatBool(options.Teardown))

	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}

	if err := resp.CheckStatus(http.StatusNoContent); err != nil {
		return WithStack(err)
	}

	return nil
}
