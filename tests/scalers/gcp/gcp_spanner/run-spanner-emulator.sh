#!/usr/bin/env bash
# Starts the Cloud Spanner emulator via Podman and creates a test instance/database.
# Usage: ./scripts/run-spanner-emulator.sh [stop]

set -euo pipefail

CONTAINER_NAME="spanner-emulator"
GRPC_PORT=9010
HTTP_PORT=9020
PROJECT="test-project"
INSTANCE="test-instance"
DATABASE="test-db"

if [[ "${1:-}" == "stop" ]]; then
  podman stop "$CONTAINER_NAME" 2>/dev/null || true
  podman rm   "$CONTAINER_NAME" 2>/dev/null || true
  echo "Spanner emulator stopped."
  exit 0
fi

if podman ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
  echo "Emulator already running."
else
  REGISTRY_AUTH_FILE=/dev/null podman run -d \
    --name "$CONTAINER_NAME" \
    -p "${GRPC_PORT}:9010" \
    -p "${HTTP_PORT}:9020" \
    gcr.io/cloud-spanner-emulator/emulator
  echo "Waiting for emulator to be ready..."
  sleep 3
fi

export SPANNER_EMULATOR_HOST="localhost:${GRPC_PORT}"

echo "Creating instance '${INSTANCE}' in project '${PROJECT}'..."
gcloud spanner instances create "$INSTANCE" \
  --config=emulator-config \
  --description="Test instance" \
  --nodes=1 \
  --project="$PROJECT" \
  --quiet 2>/dev/null || echo "Instance already exists."

echo "Creating database '${DATABASE}'..."
gcloud spanner databases create "$DATABASE" \
  --instance="$INSTANCE" \
  --project="$PROJECT" \
  --quiet 2>/dev/null || echo "Database already exists."

echo ""
echo "Spanner emulator ready."
echo "  SPANNER_EMULATOR_HOST=localhost:${GRPC_PORT}"
echo "  Project:  ${PROJECT}"
echo "  Instance: ${INSTANCE}"
echo "  Database: ${DATABASE}"
echo ""
echo "Export for your shell:"
echo "  export SPANNER_EMULATOR_HOST=localhost:${GRPC_PORT}"
