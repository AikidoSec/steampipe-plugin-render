#!/usr/bin/env bash

# Regenerate the Render API client from the public-api-schema.
#
# Set PUBLIC_API_SCHEMA_PATH to point at a local checkout of
# https://github.com/renderinc/public-api-schema. Defaults to
# ../public-api-schema relative to this repo.

set -o errexit -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCHEMA_PATH="${PUBLIC_API_SCHEMA_PATH:-$SCRIPT_DIR/../public-api-schema}"

if [[ ! -f "$SCHEMA_PATH/src/ga-schema.yaml" ]]; then
  echo "Could not find ga-schema.yaml at $SCHEMA_PATH/src/ga-schema.yaml"
  echo "Set PUBLIC_API_SCHEMA_PATH to a local checkout of renderinc/public-api-schema."
  exit 1
fi

if ! command -v oapi-codegen &>/dev/null; then
  echo "Error: oapi-codegen is not installed or not in PATH."
  echo "Install it with 'go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest'"
  exit 1
fi

CONFIG_FILE="$SCRIPT_DIR/oapi-render.yaml"
OUTPUT_DIR="$SCRIPT_DIR/render/client"

mkdir -p "$OUTPUT_DIR"

generate_file() {
  local input_file="$1"
  local output_file="$2"
  local extra_args="$3"
  local pkg="$4"

  echo "// This file has been generated from the Render REST API schema. Do not edit it manually." > "$output_file"
  echo "// See https://github.com/renderinc/public-api-schema for details." >> "$output_file"
  echo >> "$output_file"

  if [[ -n "$pkg" ]]; then
    oapi-codegen -config "$CONFIG_FILE" -package "$pkg" $extra_args "$input_file" >> "$output_file"
  else
    oapi-codegen -config "$CONFIG_FILE" $extra_args "$input_file" >> "$output_file"
  fi
}

generate_file "$SCHEMA_PATH/src/ga-schema.yaml" "$OUTPUT_DIR/types_gen.go" "-generate types" ""
generate_file "$SCHEMA_PATH/src/ga-schema.yaml" "$OUTPUT_DIR/client_gen.go" "-generate client" ""

for f in "$SCHEMA_PATH"/src/*.yaml ; do
  filename=$(basename "$f")
  component=${filename%.yaml}

  if [[ $filename != "ga-schema.yaml" ]]; then
    mkdir -p "$OUTPUT_DIR/$component"
    generate_file "$f" "$OUTPUT_DIR/$component/${component}_gen.go" "-generate types,skip-prune" "$component"
  fi
done

echo "Generated client at $OUTPUT_DIR"
