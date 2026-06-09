package resources

import (
	_ "embed"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
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
	versions := make([]string, len(m.ClickhouseVersions))
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
