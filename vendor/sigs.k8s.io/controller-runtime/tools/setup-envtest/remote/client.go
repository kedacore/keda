// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 The Kubernetes Authors

package remote

import (
	"context"
	"crypto/md5" //nolint:gosec
	"encoding/base64"
	"encoding/json"
	"errors"
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

// Client is a basic client for fetching versions of the envtest binary archives
// from GCS.
type Client struct {
	// Bucket is the bucket to fetch from.
	Bucket string

	// Server is the GCS-like storage server
	Server string

	// Log allows us to log.
	Log logr.Logger

	// Insecure uses http for testing
	Insecure bool
}

func (c *Client) scheme() string {
	if c.Insecure {
		return "http"
	}
	return "https"
}

// ListVersions lists all available tools versions in the given bucket, along
// with supported os/arch combos and the corresponding hash.
//
// The results are sorted with newer versions first.
func (c *Client) ListVersions(ctx context.Context) ([]versions.Set, error) {
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
					MD5:      item.Hash,
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
func (c *Client) GetVersion(ctx context.Context, version versions.Concrete, platform versions.PlatformItem, out io.Writer) error {
	itemName := platform.ArchiveName(version)
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

	if platform.MD5 != "" {
		// stream in chunks to do the checksum, don't load the whole thing into
		// memory to avoid causing issues with big files.
		buf := make([]byte, 32*1024) // 32KiB, same as io.Copy
		checksum := md5.New()        //nolint:gosec
		for cont := true; cont; {
			amt, err := resp.Body.Read(buf)
			if err != nil && !errors.Is(err, io.EOF) {
				return fmt.Errorf("unable read next chunk of %s: %w", itemName, err)
			}
			if amt > 0 {
				// checksum never returns errors according to docs
				checksum.Write(buf[:amt])
				if _, err := out.Write(buf[:amt]); err != nil {
					return fmt.Errorf("unable write next chunk of %s: %w", itemName, err)
				}
			}
			cont = amt > 0 && !errors.Is(err, io.EOF)
		}

		sum := base64.StdEncoding.EncodeToString(checksum.Sum(nil))

		if sum != platform.MD5 {
			return fmt.Errorf("checksum mismatch for %s: %s (computed) != %s (reported from GCS)", itemName, sum, platform.MD5)
		}
	} else if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("unable to download %s: %w", itemName, err)
	}
	return nil
}

// FetchSum fetches the checksum for the given concrete version & platform into
// the given platform item.
func (c *Client) FetchSum(ctx context.Context, ver versions.Concrete, pl *versions.PlatformItem) error {
	itemName := pl.ArchiveName(ver)
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

	pl.MD5 = item.Hash
	return nil
}
