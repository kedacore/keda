// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 The Kubernetes Authors

package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/tools/setup-envtest/versions"
)

// objectList is the parts we need of the GCS "list-objects-in-bucket" endpoint.
type objectList struct {
	Items         []bucketObject `json:"items"`
	NextPageToken string         `json:"nextPageToken"`
}

// bucketObject is the parts we need of the GCS object metadata.
type bucketObject struct {
	Name string `json:"name"`
	Hash string `json:"md5Hash"`
}

var _ Client = &GCSClient{}

// GCSClient is a basic client for fetching versions of the envtest binary archives
// from GCS.
//
// Deprecated: This client is deprecated and will be removed soon.
// The kubebuilder GCS bucket that we use with this client might be shutdown at any time,
// see: https://github.com/kubernetes/k8s.io/issues/2647.
type GCSClient struct {
	// Bucket is the bucket to fetch from.
	Bucket string

	// Server is the GCS-like storage server
	Server string

	// Log allows us to log.
	Log logr.Logger

	// Insecure uses http for testing
	Insecure bool
}

func (c *GCSClient) scheme() string {
	if c.Insecure {
		return "http"
	}
	return "https"
}

// ListVersions lists all available tools versions in the given bucket, along
// with supported os/arch combos and the corresponding hash.
//
// The results are sorted with newer versions first.
func (c *GCSClient) ListVersions(ctx context.Context) ([]versions.Set, error) {
	loc := &url.URL{
		Scheme: c.scheme(),
		Host:   c.Server,
		Path:   path.Join("/storage/v1/b/", c.Bucket, "o"),
	}
	query := make(url.Values)

	knownVersions := map[versions.Concrete][]versions.PlatformItem{}
	for cont := true; cont; {
		c.Log.V(1).Info("listing bucket to get versions", "bucket", c.Bucket)

		loc.RawQuery = query.Encode()
		req, err := http.NewRequestWithContext(ctx, "GET", loc.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("unable to construct request to list bucket items: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("unable to perform request to list bucket items: %w", err)
		}

		err = func() error {
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				return fmt.Errorf("unable list bucket items -- got status %q from GCS", resp.Status)
			}

			var list objectList
			if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
				return fmt.Errorf("unable unmarshal bucket items list: %w", err)
			}

			// continue listing if needed
			cont = list.NextPageToken != ""
			query.Set("pageToken", list.NextPageToken)

			for _, item := range list.Items {
				ver, details := versions.ExtractWithPlatform(versions.ArchiveRE, item.Name)
				if ver == nil {
					c.Log.V(1).Info("skipping bucket object -- does not appear to be a versioned tools object", "name", item.Name)
					continue
				}
				c.Log.V(1).Info("found version", "version", ver, "platform", details)
				knownVersions[*ver] = append(knownVersions[*ver], versions.PlatformItem{
					Platform: details,
					Hash: &versions.Hash{
						Type:     versions.MD5HashType,
						Encoding: versions.Base64HashEncoding,
						Value:    item.Hash,
					},
				})
			}

			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	res := make([]versions.Set, 0, len(knownVersions))
	for ver, details := range knownVersions {
		res = append(res, versions.Set{Version: ver, Platforms: details})
	}
	// sort in inverse order so that the newest one is first
	sort.Slice(res, func(i, j int) bool {
		first, second := res[i].Version, res[j].Version
		return first.NewerThan(second)
	})

	return res, nil
}

// GetVersion downloads the given concrete version for the given concrete platform, writing it to the out.
func (c *GCSClient) GetVersion(ctx context.Context, version versions.Concrete, platform versions.PlatformItem, out io.Writer) error {
	itemName := platform.ArchiveName(true, version)
	loc := &url.URL{
		Scheme:   c.scheme(),
		Host:     c.Server,
		Path:     path.Join("/storage/v1/b/", c.Bucket, "o", itemName),
		RawQuery: "alt=media",
	}

	req, err := http.NewRequestWithContext(ctx, "GET", loc.String(), nil)
	if err != nil {
		return fmt.Errorf("unable to construct request to fetch %s: %w", itemName, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to fetch %s (%s): %w", itemName, req.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unable fetch %s (%s) -- got status %q from GCS", itemName, req.URL, resp.Status)
	}

	return readBody(resp, out, itemName, platform)
}

// FetchSum fetches the checksum for the given concrete version & platform into
// the given platform item.
func (c *GCSClient) FetchSum(ctx context.Context, ver versions.Concrete, pl *versions.PlatformItem) error {
	itemName := pl.ArchiveName(true, ver)
	loc := &url.URL{
		Scheme: c.scheme(),
		Host:   c.Server,
		Path:   path.Join("/storage/v1/b/", c.Bucket, "o", itemName),
	}

	req, err := http.NewRequestWithContext(ctx, "GET", loc.String(), nil)
	if err != nil {
		return fmt.Errorf("unable to construct request to fetch metadata for %s: %w", itemName, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to fetch metadata for %s: %w", itemName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unable fetch metadata for %s -- got status %q from GCS", itemName, resp.Status)
	}

	var item bucketObject
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return fmt.Errorf("unable to unmarshal metadata for %s: %w", itemName, err)
	}

	pl.Hash = &versions.Hash{
		Type:     versions.MD5HashType,
		Encoding: versions.Base64HashEncoding,
		Value:    item.Hash,
	}
	return nil
}
