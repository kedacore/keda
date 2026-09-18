# Copyright 2026 The KEDA Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import tempfile
import unittest
from pathlib import Path

from verify_go_versions import verify


class VerifyGoVersionsTest(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name)
        self.write("go.mod", "module example.com/test\n\ngo 1.26.0\n")
        self.write(".devcontainer/Dockerfile", "FROM golang:1.26.8\n")
        self.write("Dockerfile", "FROM --platform=$BUILDPLATFORM ghcr.io/kedacore/keda-tools:1.26.7@sha256:abc AS builder\n")
        self.write(".github/workflows/ci.yml", "container: ghcr.io/kedacore/keda-tools:1.26.6\n")

    def write(self, name, content):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)

    def test_patch_versions_can_differ(self):
        self.assertEqual(verify(self.root), [])

    def test_checks_every_image_location(self):
        for name in [".devcontainer/Dockerfile", "Dockerfile", "Dockerfile.adapter", "Dockerfile.webhooks",
                     ".github/workflows/ci.yml", ".github/workflows/extra.yaml"]:
            with self.subTest(name=name):
                path = self.root / name
                original = path.read_text() if path.exists() else None
                self.write(name, "FROM ghcr.io/kedacore/keda-tools:1.27.1\n")
                self.assertEqual(verify(self.root), [f"{name}:1: ghcr.io/kedacore/keda-tools:1.27.1 uses Go 1.27; go.mod requires 1.26"])
                if original is None:
                    path.unlink()
                else:
                    path.write_text(original)

    def test_original_devcontainer_mismatch(self):
        self.write(".devcontainer/Dockerfile", "FROM golang:1.27.1\n")
        self.assertEqual(verify(self.root), [".devcontainer/Dockerfile:1: golang:1.27.1 uses Go 1.27; go.mod requires 1.26"])

    def test_all_references_in_workflow_are_checked(self):
        self.write(".github/workflows/ci.yml", "container: ghcr.io/kedacore/keda-tools:1.26.8\ncontainer: ghcr.io/kedacore/keda-tools:1.25.8\n")
        self.assertEqual(verify(self.root), [".github/workflows/ci.yml:2: ghcr.io/kedacore/keda-tools:1.25.8 uses Go 1.25; go.mod requires 1.26"])

    def test_major_version_mismatch(self):
        self.write(".devcontainer/Dockerfile", "FROM golang:2.26.0\n")
        self.assertEqual(verify(self.root), [".devcontainer/Dockerfile:1: golang:2.26.0 uses Go 2.26; go.mod requires 1.26"])

    def test_image_suffix_and_digest(self):
        self.write(".devcontainer/Dockerfile", "FROM golang:1.26.8-bookworm@sha256:abc\n")
        self.assertEqual(verify(self.root), [])

    def test_comments_and_unrelated_images_are_ignored(self):
        self.write(".github/workflows/extra.yaml", "  # image: golang:1.27.1\ncontainer: ubuntu:24.04\n")
        self.assertEqual(verify(self.root), [])

    def test_go_directive_without_patch(self):
        self.write("go.mod", "module example.com/test\ngo 1.26\n")
        self.assertEqual(verify(self.root), [])

    def test_missing_go_directive(self):
        self.write("go.mod", "module example.com/test\n")
        self.assertEqual(verify(self.root), ["go.mod: missing or invalid Go version directive"])

    def test_unversioned_image_is_rejected(self):
        self.write(".devcontainer/Dockerfile", "FROM golang:latest\n")
        self.assertEqual(verify(self.root), [".devcontainer/Dockerfile:1: cannot determine Go release from golang:latest"])

    def test_missing_image_is_rejected(self):
        self.write(".devcontainer/Dockerfile", "# FROM golang:1.26.8\n")
        self.assertEqual(verify(self.root), [".devcontainer/Dockerfile: no versioned golang or keda-tools image found"])


if __name__ == "__main__":
    unittest.main()
