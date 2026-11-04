#!/usr/bin/env python3

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

"""Check that development, build, and CI Go releases match go.mod."""

import re
import sys
from pathlib import Path


IMAGE = re.compile(r"\b(golang|ghcr\.io/kedacore/keda-tools)(?::([^\s@\"']+)|@[^\s\"']+)")
VERSION = re.compile(r"(\d+\.\d+)(?:\.\d+)?(?:-[\w.-]+)?")


def verify(root: Path) -> list[str]:
    directive = re.search(r"^go\s+(\d+\.\d+)(?:\.\d+)?\s*$", (root / "go.mod").read_text(), re.MULTILINE)
    if directive is None:
        return ["go.mod: missing or invalid Go version directive"]
    expected = directive.group(1)
    errors = []
    devcontainer = root / ".devcontainer/Dockerfile"
    files = sorted(root.glob("Dockerfile*")) + [devcontainer]
    files += sorted((root / ".github/workflows").glob("*.yml"))
    files += sorted((root / ".github/workflows").glob("*.yaml"))
    for path in files:
        found = False
        for number, line in enumerate(path.read_text().splitlines(), 1):
            if line.lstrip().startswith("#"):
                continue
            for image in IMAGE.finditer(line):
                found = True
                tag = image.group(2)
                version = VERSION.fullmatch(tag) if tag is not None else None
                location = f"{path.relative_to(root)}:{number}"
                if version is None:
                    errors.append(f"{location}: cannot determine Go release from {image.group(0)}")
                elif version.group(1) != expected:
                    errors.append(f"{location}: {image.group(0)} uses Go {version.group(1)}; go.mod requires {expected}")
        # A missing image must not silently bypass validation in a Dockerfile.
        # Workflows without Go images are expected (e.g. setup-go reads go.mod).
        if not found and (path == devcontainer or path.parent == root):
            errors.append(f"{path.relative_to(root)}: no versioned golang or keda-tools image found")
    return errors


def main() -> int:
    try:
        errors = verify(Path(__file__).resolve().parent.parent)
    except OSError as error:
        print(f"Cannot check Go versions: {error}", file=sys.stderr)
        return 1
    if errors:
        print("Go toolchain versions are not aligned:", file=sys.stderr)
        for error in errors:
            print(f"  {error}", file=sys.stderr)
        print("Update go.mod, the devcontainer, and build/CI images together; patch versions may differ.", file=sys.stderr)
        return 1
    print("Go major/minor versions match go.mod across development, build, and CI images.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
