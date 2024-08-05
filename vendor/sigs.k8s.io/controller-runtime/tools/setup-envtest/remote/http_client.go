// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 The Kubernetes Authors

package remote

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/tools/setup-envtest/versions"
	"sigs.k8s.io/yaml"
)

// DefaultIndexURL is the default index used in HTTPClient.
var DefaultIndexURL = "https://raw.githubusercontent.com/kubernetes-sigs/controller-tools/HEAD/envtest-releases.yaml"

var _ Client = &HTTPClient{}

// HTTPClient is a client for fetching versions of the envtest binary archives
// from an index via HTTP.
type HTTPClient struct {
	// Log allows us to log.
	Log logr.Logger

	// IndexURL is the URL of the index, defaults to DefaultIndexURL.
	IndexURL string
}

// Index represents an index of envtest binary archives. Example:
//
//	releases:
//		v1.28.0:
//			envtest-v1.28.0-darwin-amd64.tar.gz:
//	    		hash: <sha512-hash>
//				selfLink: <url-to-archive-with-envtest-binaries>
type Index struct {
	// Releases maps Kubernetes versions to Releases (envtest archives).
	Releases map[string]Release `json:"releases"`
}

// Release maps an archive name to an archive.
type Release map[string]Archive

// Archive contains the self link to an archive and its hash.
type Archive struct {
	Hash     string `json:"hash"`
	SelfLink string `json:"selfLink"`
}

// ListVersions lists all available tools versions in the index, along
// with supported os/arch combos and the corresponding hash.
//
// The results are sorted with newer versions first.
func (c *HTTPClient) ListVersions(ctx context.Context) ([]versions.Set, error) {
	index, err := c.getIndex(ctx)
	if err != nil {
		return nil, err
	}

	knownVersions := map[versions.Concrete][]versions.PlatformItem{}
	for _, releases := range index.Releases {
		for archiveName, archive := range releases {
			ver, details := versions.ExtractWithPlatform(versions.ArchiveRE, archiveName)
			if ver == nil {
				c.Log.V(1).Info("skipping archive -- does not appear to be a versioned tools archive", "name", archiveName)
				continue
			}
			c.Log.V(1).Info("found version", "version", ver, "platform", details)
			knownVersions[*ver] = append(knownVersions[*ver], versions.PlatformItem{
				Platform: details,
				Hash: &versions.Hash{
					Type:     versions.SHA512HashType,
					Encoding: versions.HexHashEncoding,
					Value:    archive.Hash,
				},
			})
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
func (c *HTTPClient) GetVersion(ctx context.Context, version versions.Concrete, platform versions.PlatformItem, out io.Writer) error {
	index, err := c.getIndex(ctx)
	if err != nil {
		return err
	}

	var loc *url.URL
	var name string
	for _, releases := range index.Releases {
		for archiveName, archive := range releases {
			ver, details := versions.ExtractWithPlatform(versions.ArchiveRE, archiveName)
			if ver == nil {
				c.Log.V(1).Info("skipping archive -- does not appear to be a versioned tools archive", "name", archiveName)
				continue
			}

			if *ver == version && details.OS == platform.OS && details.Arch == platform.Arch {
				loc, err = url.Parse(archive.SelfLink)
				if err != nil {
					return fmt.Errorf("error parsing selfLink %q, %w", loc, err)
				}
				name = archiveName
				break
			}
		}
	}
	if name == "" {
		return fmt.Errorf("unable to find archive for %s (%s,%s)", version, platform.OS, platform.Arch)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", loc.String(), nil)
	if err != nil {
		return fmt.Errorf("unable to construct request to fetch %s: %w", name, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to fetch %s (%s): %w", name, req.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unable fetch %s (%s) -- got status %q", name, req.URL, resp.Status)
	}

	return readBody(resp, out, name, platform)
}

// FetchSum fetches the checksum for the given concrete version & platform into
// the given platform item.
func (c *HTTPClient) FetchSum(ctx context.Context, version versions.Concrete, platform *versions.PlatformItem) error {
	index, err := c.getIndex(ctx)
	if err != nil {
		return err
	}

	for _, releases := range index.Releases {
		for archiveName, archive := range releases {
			ver, details := versions.ExtractWithPlatform(versions.ArchiveRE, archiveName)
			if ver == nil {
				c.Log.V(1).Info("skipping archive -- does not appear to be a versioned tools archive", "name", archiveName)
				continue
			}

			if *ver == version && details.OS == platform.OS && details.Arch == platform.Arch {
				platform.Hash = &versions.Hash{
					Type:     versions.SHA512HashType,
					Encoding: versions.HexHashEncoding,
					Value:    archive.Hash,
				}
				return nil
			}
		}
	}

	return fmt.Errorf("unable to find archive for %s (%s,%s)", version, platform.OS, platform.Arch)
}

func (c *HTTPClient) getIndex(ctx context.Context) (*Index, error) {
	indexURL := c.IndexURL
	if indexURL == "" {
		indexURL = DefaultIndexURL
	}

	loc, err := url.Parse(indexURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse index URL: %w", err)
	}

	c.Log.V(1).Info("listing versions", "index", indexURL)

	req, err := http.NewRequestWithContext(ctx, "GET", loc.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("unable to construct request to get index: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to perform request to get index: %w", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unable to get index -- got status %q", resp.Status)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to get index -- unable to read body %w", err)
	}

	var index Index
	if err := yaml.Unmarshal(responseBody, &index); err != nil {
		return nil, fmt.Errorf("unable to unmarshal index: %w", err)
	}
	return &index, nil
}
