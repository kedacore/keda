#!/bin/sh
set -euo pipefail

# Check scalers in the registry file
SCALERS_FILE="pkg/scaling/scalers_registry.go"

# Extract scaler names from RegisterScalerBuilder calls
CURRENT=$(grep -o 'RegisterScalerBuilder("[^"]*"' "${SCALERS_FILE}" | cut -d'"' -f2)
SORTED=$(grep -o 'RegisterScalerBuilder("[^"]*"' "${SCALERS_FILE}" | cut -d'"' -f2 | sort)

if [[ "${CURRENT}" == "${SORTED}" ]]; then
  echo "Scalers are sorted in ${SCALERS_FILE}"
  exit 0
else
  echo "Scalers are not sorted alphabetically in ${SCALERS_FILE}"
  exit 1
fi
