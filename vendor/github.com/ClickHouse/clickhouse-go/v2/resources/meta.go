// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package resources

import (
	_ "embed"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
	"gopkg.in/yaml.v3"
	"strings"
)

type Meta struct {
	ClickhouseVersions []proto.Version `yaml:"clickhouse_versions"`
	GoVersions         []proto.Version `yaml:"go_versions"`
	hVersion           proto.Version
}

//go:embed meta.yml
var metaFile []byte
var ClientMeta Meta

func init() {
	if err := yaml.Unmarshal(metaFile, &ClientMeta); err != nil {
		panic(err)
	}
	ClientMeta.hVersion = ClientMeta.findGreatestVersion()
}

func (m *Meta) IsSupportedClickHouseVersion(v proto.Version) bool {
	for _, version := range m.ClickhouseVersions {
		if version.Major == v.Major && version.Minor == v.Minor {
			// check our patch is greater
			return v.Patch >= version.Patch
		}
	}
	return proto.CheckMinVersion(m.hVersion, v)
}

func (m *Meta) SupportedVersions() string {
	versions := make([]string, len(m.ClickhouseVersions), len(m.ClickhouseVersions))
	for i := range m.ClickhouseVersions {
		versions[i] = m.ClickhouseVersions[i].String()
	}
	return strings.Join(versions, ", ")
}

func (m *Meta) findGreatestVersion() proto.Version {
	var maxVersion proto.Version
	for _, version := range m.ClickhouseVersions {
		if version.Major > maxVersion.Major {
			maxVersion = version
		} else if version.Major == maxVersion.Major {
			if version.Minor > maxVersion.Minor {
				maxVersion = version
			} else if version.Minor == maxVersion.Minor {
				if version.Patch > maxVersion.Patch {
					maxVersion = version
				}
			}
		}
	}
	return maxVersion
}
