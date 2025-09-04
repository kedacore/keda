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
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	adlsfs "github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/filesystem"
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

	// When a glob is present, prefer hierarchical (DFS) listing if available (HNS),
	// then fall back to flat blob listing.
	if meta.GlobPattern != nil {
		// 1) Try ADLS Gen2 DFS listing (works correctly for hierarchical namespace).
		if meta.Connection != "" {
			if cnt, handled, err := tryCountWithDFS(ctx, meta); handled {
				return cnt, err
			}
			// handled == false -> DFS not usable (likely non-HNS or missing perms); fall back below.
		}

		// 2) Fallback: flat Blob listing with client-side glob matching (works for classic Blob and most HNS cases).
		globPattern := *meta.GlobPattern
		var count int64

		flatPager := containerClient.NewListBlobsFlatPager(&azblob.ListBlobsFlatOptions{
			Prefix: meta.BlobPrefix, // perf hint only
		})

		var normPrefix string
		if meta.BlobPrefix != nil {
			normPrefix = strings.TrimPrefix(*meta.BlobPrefix, "/")
		}

		for flatPager.More() {
			resp, err := flatPager.NextPage(ctx)
			if err != nil {
				return -1, err
			}
			for _, blobItem := range resp.Segment.BlobItems {
				if blobItem.Name == nil {
					continue
				}
				full := *blobItem.Name
				tail := full
				if normPrefix != "" {
					tail = strings.TrimPrefix(tail, normPrefix)
					tail = strings.TrimPrefix(tail, "/") // ensure no leading slash after prefix removal
				}
				if globPattern.Match(full) || globPattern.Match(tail) {
					count++
				}
			}
		}
		return count, nil
	}

	// No glob -> preserve your original one-level hierarchical behavior.
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

// tryCountWithDFS counts files using ADLS Gen2 DFS listing (hierarchical).
// Returns (count, handled=true, err) if DFS was attempted; (0, handled=false, nil) if DFS couldn't be used.
func tryCountWithDFS(ctx context.Context, meta *BlobMetadata) (int64, bool, error) {
	// Build a filesystem client from the connection string; this will target the DFS endpoint
	// and gracefully fail on non-HNS accounts (allowing us to fallback).
	fs, err := adlsfs.NewClientFromConnectionString(meta.Connection, meta.BlobContainerName, nil)
	if err != nil {
		// Connection string missing parts or not suitable -> let caller fallback.
		return 0, false, nil
	}

	var prefix string
	if meta.BlobPrefix != nil {
		prefix = strings.TrimPrefix(*meta.BlobPrefix, "/")
	}

	var count int64
	pager := fs.NewListPathsPager(true, &adlsfs.ListPathsOptions{
		Prefix: &prefix, // hierarchical filter
	})
	gl := *meta.GlobPattern

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			// If DFS listing fails (e.g., non-HNS account, auth not allowed), tell caller to fallback.
			return 0, false, nil
		}
		for _, p := range resp.Paths {
			// Skip directories; we only count files (blobs).
			if p.IsDirectory != nil && *p.IsDirectory {
				continue
			}
			name := ""
			if p.Name != nil {
				name = *p.Name // path relative to filesystem
			}
			// Tail relative to prefix for user-friendly globbing.
			tail := name
			if prefix != "" {
				tail = strings.TrimPrefix(tail, prefix)
				tail = strings.TrimPrefix(tail, "/")
			}
			if gl.Match(name) || gl.Match(tail) {
				count++
			}
		}
	}
	return count, true, nil
}
