#!/bin/bash

# Generate Swagger API documentation
# Requires: go install github.com/swaggo/swag/cmd/swag@latest

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "📄 Generating API documentation..."

swag init \
  --generalInfo internal/api/rest/router.go \
  --dir "${PROJECT_ROOT}" \
  --output "${PROJECT_ROOT}/docs" \
  --parseDependency \
  --parseInternal

echo "✅ API documentation generated at: ${PROJECT_ROOT}/docs/"
echo "📖 Access docs at: {base_url}/docs/api/index.html"
