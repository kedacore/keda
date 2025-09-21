#!/bin/sh
set -euo pipefail

LEAD='TRIGGERS-START'
TAIL='TRIGGERS-END'

SCALERS_FILE="pkg/scaling/scalers_builder.go"
CURRENT=$(cat "${SCALERS_FILE}" | awk "/${LEAD}/,/${TAIL}/" | grep "case")
SORTED=$(cat "${SCALERS_FILE}" | awk "/${LEAD}/,/${TAIL}/" | grep "case" | sort)

if [[ "${CURRENT}" == "${SORTED}" ]]; then
  echo "Scalers are sorted in ${SCALERS_FILE}"
  exit 0
else
  echo "Scalers are not sorted alphabetical in ${SCALERS_FILE}"
  exit 1
fi
