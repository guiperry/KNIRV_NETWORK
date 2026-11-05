#!/usr/bin/env bash
set -euo pipefail

server_dir=$(cd "$(dirname "$0")/.." && pwd)
plugin_dir="$server_dir/pkg/embedded/knirvbao/plugin"
output_dir="$server_dir/pkg/embedded/knirvbao/bin"
source_version=2.7.2
source_sha256=11b48e34ea1bbad5db4044ced904c7f1a4265da5230551463846c6745e507797
go_version=$(tr -d '[:space:]' < "$server_dir/.go-version")
scratch=$(mktemp -d "/tmp/knirvbao-build.XXXXXXXX")
trap 'rm -rf "$scratch"' EXIT

archive=${OPENBAO_SOURCE_ARCHIVE:-$scratch/openbao.tar.gz}
if [[ -z ${OPENBAO_SOURCE_ARCHIVE:-} ]]; then
  curl -fL "https://github.com/openbao/openbao/archive/refs/tags/v${source_version}.tar.gz" -o "$archive"
fi
printf '%s  %s\n' "$source_sha256" "$archive" | sha256sum -c -
tar -xzf "$archive" -C "$scratch"
source_dir="$scratch/openbao-$source_version"
if [[ $(tr -d '[:space:]' < "$source_dir/.go-version") != "$go_version" ]]; then
  echo "OpenBao source and KNIRVSERVER Go pins differ" >&2
  exit 1
fi

# This also checks the embedded SDK core against the current SDK stamp.
(cd "$plugin_dir" && env -u GOROOT GOTOOLCHAIN="go$go_version" CGO_ENABLED=0 go test -run '^TestEmbeddedCoreMatchesSDKPin$' ./...)
(cd "$source_dir" && env -u GOROOT GOTOOLCHAIN="go$go_version" CGO_ENABLED=0 go build -trimpath -ldflags "-X github.com/openbao/openbao/v2/internal/version.fullVersion=$source_version" -o "$scratch/bao" .)
if [[ $("$scratch/bao" version) != "OpenBao v$source_version"* ]]; then
  echo "built OpenBao reports the wrong version" >&2
  exit 1
fi
(cd "$plugin_dir" && env -u GOROOT GOTOOLCHAIN="go$go_version" CGO_ENABLED=0 go build -trimpath -o "$scratch/openbao-plugin-secrets-knirv" .)
mkdir -p "$output_dir"
install -m 0755 "$scratch/bao" "$output_dir/bao"
install -m 0755 "$scratch/openbao-plugin-secrets-knirv" "$output_dir/openbao-plugin-secrets-knirv"
(cd "$output_dir" && sha256sum bao openbao-plugin-secrets-knirv > SHA256SUMS)
cat "$output_dir/SHA256SUMS"
