#!/usr/bin/env bash

set -euo pipefail

version="${1:?usage: build-release.sh <version>}"
if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
	printf 'Invalid release version: %s\n' "$version" >&2
	exit 1
fi

mkdir -p bin
ldflags="-s -w -X github.com/PedroHercules/gommit/pkg/interfaces/cli.version=$version"

build() {
	local goos="$1"
	local goarch="$2"
	local extension="$3"
	local output="bin/gmit-${goos}-${goarch}${extension}"

	printf 'Building %s/%s (%s)\n' "$goos" "$goarch" "$version"
	CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build \
		-trimpath \
		-ldflags "$ldflags" \
		-o "$output" .
	if [[ "$goos" != "windows" ]]; then
		chmod 755 "$output"
	fi
}

build linux amd64 ""
build linux arm64 ""
build darwin amd64 ""
build darwin arm64 ""
build windows amd64 .exe
build windows arm64 .exe
