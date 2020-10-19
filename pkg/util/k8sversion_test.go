package util

import (
	"testing"

	"k8s.io/apimachinery/pkg/version"
)

type testMetadata struct {
	comment        string
	expectedMinor  int
	expectedPretty string
	version        *version.Info
}

var testMetadatas = []testMetadata{
	{
		comment:        "Testing 1.18+",
		expectedMinor:  18,
		expectedPretty: "1.18+",
		version: &version.Info{
			Major: "1",
			Minor: "18+",
		},
	},
	{
		comment:        "Testing 1.18",
		expectedMinor:  18,
		expectedPretty: "1.18",
		version: &version.Info{
			Major: "1",
			Minor: "18",
		},
	},
	{
		comment:        "Testing 1.19.84324.2",
		expectedMinor:  19,
		expectedPretty: "1.19.84324.2",
		version: &version.Info{
			Major: "1",
			Minor: "19.84324.2",
		},
	},
	{
		comment:        "Testing 2.1",
		expectedMinor:  1,
		expectedPretty: "2.1",
		version: &version.Info{
			Major: "2",
			Minor: "1",
		},
	},
	{
		comment:        "Testing 2.",
		expectedMinor:  0,
		expectedPretty: "2.",
		version: &version.Info{
			Major: "2",
			Minor: "",
		},
	},
}

func TestResolveK8sVersion(t *testing.T) {
	for _, testData := range testMetadatas {
		t.Log(testData.comment)

		version := NewK8sVersion(testData.version)

		if version.MinorVersion != testData.expectedMinor {
			t.Error("Failed to resolve k8s Minor Version correctly", "wants", testData.expectedMinor, "got", version.MinorVersion)
		}

		if version.PrettyVersion != testData.expectedPretty {
			t.Error("Failed to resolve k8s Pretty Version correctly", "wants", testData.expectedPretty, "got", version.PrettyVersion)
		}
	}
}
