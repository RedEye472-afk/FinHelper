#!/bin/bash
# Usage: ./scripts/download-file.sh <url> [output-path]
url="$1"
output="${2:-$(basename "$url")}"
curl -L --retry 3 --retry-delay 2 -o "$output" "$url"
echo "Downloaded to: $output"
