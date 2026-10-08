#!/bin/sh
set -eu
cd "$(dirname "$0")"
podman build --network=host --platform linux/arm64 \
  --build-arg BUILDPLATFORM=linux/amd64 \
  --build-arg TARGETOS=linux --build-arg TARGETARCH=arm64 \
  -t localhost/warpscout-mikrotik:arm64 .
podman image inspect localhost/warpscout-mikrotik:arm64 --format '{{.Os}}/{{.Architecture}}'
archive_tmp="warpscout-mikrotik-arm64.$$.tar"
podman save --format docker-archive --output "$archive_tmp" localhost/warpscout-mikrotik:arm64
mv -f "$archive_tmp" warpscout-mikrotik-arm64.tar
sha256sum warpscout-mikrotik-arm64.tar
