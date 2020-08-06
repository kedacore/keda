package scalers

import "testing"

var testAzBlobResolvedEnv = map[string]string{
	"CONNECTION": "SAMPLE",
}

type parseAzBlobMetadataTestData struct {
	metadata    map[string]string
	isError     bool
	resolvedEnv map[string]string
	authParams  map[string]string
	podIdentity string
}

var testAzBlobMetadata = []parseAzBlobMetadataTestData{
	// nothing passed
	{map[string]string{}, true, testAzBlobResolvedEnv, map[string]string{}, ""},
	// properly formed
	{map[string]string{"connection": "CONNECTION", "blobContainerName": "sample", "blobCount": "5", "blobDelimiter": "/", "blobPrefix": "blobsubpath"}, false, testAzBlobResolvedEnv, map[string]string{}, ""},
	// Empty blobcontainerName
	{map[string]string{"connection": "CONNECTION", "blobContainerName": ""}, true, testAzBlobResolvedEnv, map[string]string{}, ""},
	// improperly formed blobCount
	{map[string]string{"connection": "CONNECTION", "blobContainerName": "sample", "blobCount": "AA"}, true, testAzBlobResolvedEnv, map[string]string{}, ""},
	// podIdentity = azure with account name
	{map[string]string{"accountName": "sample_acc", "blobContainerName": "sample_container"}, false, testAzBlobResolvedEnv, map[string]string{}, "azure"},
	// podIdentity = azure without account name
	{map[string]string{"accountName": "", "blobContainerName": "sample_container"}, true, testAzBlobResolvedEnv, map[string]string{}, "azure"},
	// podIdentity = azure without blob container name
	{map[string]string{"accountName": "sample_acc", "blobContainerName": ""}, true, testAzBlobResolvedEnv, map[string]string{}, "azure"},
	// connection from authParams
	{map[string]string{"blobContainerName": "sample_container", "blobCount": "5"}, false, testAzBlobResolvedEnv, map[string]string{"connection": "value"}, "none"},
}

func TestAzBlobParseMetadata(t *testing.T) {
	for _, testData := range testAzBlobMetadata {
		_, podIdentity, err := parseAzureBlobMetadata(testData.metadata, testData.resolvedEnv, testData.authParams, testData.podIdentity)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success. testData: %v", testData)
		}
		if testData.podIdentity != "" && testData.podIdentity != podIdentity && err == nil {
			t.Error("Expected success but got error: podIdentity value is not returned as expected")

		}
	}
}
