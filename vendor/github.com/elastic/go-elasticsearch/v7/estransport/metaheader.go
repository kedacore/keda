// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package estransport

import (
	"regexp"
	"runtime"
	"strings"
)

// HeaderClientMeta Key for the HTTP Header related to telemetry data sent with
// each request to Elasticsearch.
const HeaderClientMeta = "x-elastic-client-meta"

var metaReVersion = regexp.MustCompile("([0-9.]+)(.*)")

func initMetaHeader() string {
	var b strings.Builder
	var strippedGoVersion string
	var strippedEsVersion string

	strippedEsVersion = buildStrippedVersion(Version)
	strippedGoVersion = buildStrippedVersion(runtime.Version())

	var duos = [][]string{
		{
			"es",
			strippedEsVersion,
		},
		{
			"go",
			strippedGoVersion,
		},
		{
			"t",
			strippedEsVersion,
		},
		{
			"hc",
			strippedGoVersion,
		},
	}

	var arr []string
	for _, duo := range duos {
		arr = append(arr, strings.Join(duo, "="))
	}
	b.WriteString(strings.Join(arr, ","))

	return b.String()
}

func buildStrippedVersion(version string) string {
	v := metaReVersion.FindStringSubmatch(version)

	if len(v) == 3 && !strings.Contains(version, "devel") {
		switch {
		case v[2] != "":
			return v[1] + "p"
		default:
			return v[1]
		}
	}

	return "0.0p"
}
