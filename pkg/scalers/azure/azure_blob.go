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

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/gobwas/glob"
)

type BlobMetadata struct {
	TargetBlobCount           int64
	ActivationTargetBlobCount int64
	BlobContainerName         string
	BlobDelimiter             string
	BlobPrefix                *string
	Connection                string
	AccountName               string
	EndpointSuffix            string
	TriggerIndex              int
	GlobPattern               *glob.Glob
}

// GetAzureBlobListLength returns the count of the blobs in blob container in int
func GetAzureBlobListLength(ctx context.Context, blobClient *azblob.Client, meta *BlobMetadata) (int64, error) {
	containerClient := blobClient.ServiceClient().NewContainerClient(meta.BlobContainerName)
	if meta.GlobPattern != nil {
		globPattern := *meta.GlobPattern
		var count int64
		flatPager := containerClient.NewListBlobsFlatPager(&azblob.ListBlobsFlatOptions{
			Prefix: meta.BlobPrefix,
		})
		for flatPager.More() {
			resp, err := flatPager.NextPage(ctx)
			if err != nil {
				return -1, err
			}
			for _, blobItem := range resp.Segment.BlobItems {
				if blobItem.Name != nil && globPattern.Match(*blobItem.Name) {
					count++
				}
			}
		}
		return count, nil
	}
	hierarchyPager := containerClient.NewListBlobsHierarchyPager(meta.BlobDelimiter, &container.ListBlobsHierarchyOptions{
		Prefix: meta.BlobPrefix,
	})
	var count int64
	for hierarchyPager.More() {
		resp, err := hierarchyPager.NextPage(ctx)
		if err != nil {
			return -1, err
		}
		count += int64(len(resp.Segment.BlobItems))
	}
	return count, nil
}
