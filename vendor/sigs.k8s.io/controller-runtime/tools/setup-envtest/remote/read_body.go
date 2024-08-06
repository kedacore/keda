// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 The Kubernetes Authors

package remote

import (
	//nolint:gosec // We're aware that md5 is a weak cryptographic primitive, but we don't have a choice here.
	"crypto/md5"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"

	"sigs.k8s.io/controller-runtime/tools/setup-envtest/versions"
)

func readBody(resp *http.Response, out io.Writer, archiveName string, platform versions.PlatformItem) error {
	if platform.Hash != nil {
		// stream in chunks to do the checksum, don't load the whole thing into
		// memory to avoid causing issues with big files.
		buf := make([]byte, 32*1024) // 32KiB, same as io.Copy
		var hasher hash.Hash
		switch platform.Hash.Type {
		case versions.SHA512HashType:
			hasher = sha512.New()
		case versions.MD5HashType:
			hasher = md5.New() //nolint:gosec // We're aware that md5 is a weak cryptographic primitive, but we don't have a choice here.
		default:
			return fmt.Errorf("hash type %s not implemented", platform.Hash.Type)
		}
		for cont := true; cont; {
			amt, err := resp.Body.Read(buf)
			if err != nil && !errors.Is(err, io.EOF) {
				return fmt.Errorf("unable read next chunk of %s: %w", archiveName, err)
			}
			if amt > 0 {
				// checksum never returns errors according to docs
				hasher.Write(buf[:amt])
				if _, err := out.Write(buf[:amt]); err != nil {
					return fmt.Errorf("unable write next chunk of %s: %w", archiveName, err)
				}
			}
			cont = amt > 0 && !errors.Is(err, io.EOF)
		}

		var sum string
		switch platform.Hash.Encoding {
		case versions.Base64HashEncoding:
			sum = base64.StdEncoding.EncodeToString(hasher.Sum(nil))
		case versions.HexHashEncoding:
			sum = hex.EncodeToString(hasher.Sum(nil))
		default:
			return fmt.Errorf("hash encoding %s not implemented", platform.Hash.Encoding)
		}
		if sum != platform.Hash.Value {
			return fmt.Errorf("checksum mismatch for %s: %s (computed) != %s (reported)", archiveName, sum, platform.Hash.Value)
		}
	} else if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("unable to download %s: %w", archiveName, err)
	}
	return nil
}
