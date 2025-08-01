//go:build go1.24

package internal

import (
	"crypto/fips140"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

// callers MUST hold binaryChecksumLock before calling
func initBinaryChecksumLocked() error {
	if len(binaryChecksum) > 0 {
		return nil
	}

	exec, err := os.Executable()
	if err != nil {
		return err
	}

	f, err := os.Open(exec)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close() // error is unimportant as it is read-only
	}()

	var h hash.Hash
	if fips140.Enabled() {
		h = sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
	} else {
		h = md5.New()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
	}

	checksum := h.Sum(nil)
	binaryChecksum = hex.EncodeToString(checksum[:])

	return nil
}
