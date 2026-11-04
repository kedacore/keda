#!/usr/bin/env bash
# Fails if a file under tests/ contains time.Sleep and is not listed in
# tools/test_sleep_allowlist.txt. The shared wait helpers (KedaEventually,
# KedaConsistently, and the WaitFor* wrappers built on them) are the supported
# way to wait in e2e tests — see tests/README.md.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ALLOWLIST="${ROOT}/tools/test_sleep_allowlist.txt"

if [[ ! -f "${ALLOWLIST}" ]]; then
  echo "missing allowlist: ${ALLOWLIST}" >&2
  exit 1
fi

mapfile -t allowed < <(grep -vE '^\s*(#|$)' "${ALLOWLIST}" | sort -u)

# Paths relative to the repo root, one per file that still uses time.Sleep.
mapfile -t offenders < <(
  cd "${ROOT}"
  # -I skips binary matches; restrict to Go sources under tests/.
  grep -RIl --include='*.go' 'time\.Sleep' tests/ | sort -u
)

failed=0
for f in "${offenders[@]+"${offenders[@]}"}"; do
  ok=0
  for a in "${allowed[@]+"${allowed[@]}"}"; do
    if [[ "${f}" == "${a}" ]]; then
      ok=1
      break
    fi
  done
  if [[ "${ok}" -eq 0 ]]; then
    echo "disallowed time.Sleep in ${f}" >&2
    echo "  use KedaEventually / KedaConsistently (tests/helper), or add the file to tools/test_sleep_allowlist.txt with a reason in #7991" >&2
    failed=1
  fi
done

# Flag allowlist entries that no longer contain a sleep so the list can shrink.
for a in "${allowed[@]+"${allowed[@]}"}"; do
  if [[ ! -f "${ROOT}/${a}" ]]; then
    echo "allowlist entry missing on disk (remove it): ${a}" >&2
    failed=1
    continue
  fi
  if ! grep -q 'time\.Sleep' "${ROOT}/${a}"; then
    echo "allowlist entry no longer contains time.Sleep (remove it): ${a}" >&2
    failed=1
  fi
done

if [[ "${failed}" -ne 0 ]]; then
  exit 1
fi

echo "No new time.Sleep under tests/ (${#offenders[@]} allowlisted file(s))"
