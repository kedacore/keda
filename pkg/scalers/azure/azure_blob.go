/*
Copyright 2021 The KEDA Authors

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

package azure

import (
	"context"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/gobwas/glob"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type BlobMetadata struct {
	TargetBlobCount           int64
	ActivationTargetBlobCount int64
	BlobContainerName         string
	BlobDelimiter             string
	BlobPrefix                string
	Connection                string
	AccountName               string
	EndpointSuffix            string
	TriggerIndex              int
	GlobPattern               *glob.Glob
}

// GetAzureBlobListLength returns the count of the blobs in blob container in int
func GetAzureBlobListLength(ctx context.Context, podIdentity kedav1alpha1.AuthPodIdentity, meta *BlobMetadata) (int64, error) {
	credential, endpoint, err := ParseAzureStorageBlobConnection(ctx, podIdentity, meta.Connection, meta.AccountName, meta.EndpointSuffix)
	if err != nil {
		return -1, err
	}

	listBlobsSegmentOptions := azblob.ListBlobsSegmentOptions{
		Prefix: meta.BlobPrefix,
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	serviceURL := azblob.NewServiceURL(*endpoint, p)
	containerURL := serviceURL.NewContainerURL(meta.BlobContainerName)

	if meta.GlobPattern != nil {
		props, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{})
		if err != nil {
			return -1, err
		}

		var count int64
		globPattern := *meta.GlobPattern
		for _, blobItem := range props.Segment.BlobItems {
			if globPattern.Match(blobItem.Name) {
				count++
			}
		}
		return count, nil
	}

	props, err := containerURL.ListBlobsHierarchySegment(ctx, azblob.Marker{}, meta.BlobDelimiter, listBlobsSegmentOptions)
	if err != nil {
		return -1, err
	}

	return int64(len(props.Segment.BlobItems)), nil
}
