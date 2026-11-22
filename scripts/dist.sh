#!/bin/sh
# dist.sh builds jobwatch for each target and packs one archive each, with
# the licences of everything compiled in, and a SHA256SUMS file.
#
#   VERSION=0.2.0 scripts/dist.sh             # Linux amd64 and arm64
#   TARGETS="linux/amd64 darwin/arm64" scripts/dist.sh
#
# The archives hold: jobwatch, README.md, LICENSE, jobwatch.el, and
# THIRD_PARTY/<module>/ with each compiled-in module's LICENSE and NOTICE,
# since Apache-2.0 asks for those with a binary.
set -eu

version=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')}
targets=${TARGETS:-"linux/amd64 linux/arm64"}
out=${OUT:-dist}

rm -rf "$out"
mkdir -p "$out"

# the modules compiled into jobwatch, and where their files are
third=$(mktemp -d)
trap 'rm -rf "$third"' EXIT
go list -deps -f '{{with .Module}}{{if not .Main}}{{.Path}} {{.Dir}}{{end}}{{end}}' ./cmd/jobwatch | sort -u |
while read -r path dir; do
	[ -n "$dir" ] || continue
	found=
	for f in "$dir"/LICENSE* "$dir"/LICENCE* "$dir"/COPYING* "$dir"/NOTICE*; do
		[ -f "$f" ] || continue
		mkdir -p "$third/$path"
		cp "$f" "$third/$path/"
		found=1
	done
	if [ -z "$found" ]; then
		echo "dist: $path has no licence file" >&2
		exit 1
	fi
done

for t in $targets; do
	os=${t%/*}
	arch=${t#*/}
	name="jobwatch-$version-$os-$arch"
	dir="$out/$name"
	mkdir -p "$dir"
	GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -trimpath \
		-ldflags="-s -w -X main.Version=$version" -o "$dir/jobwatch" ./cmd/jobwatch
	cp README.md LICENSE jobwatch.el "$dir/"
	cp -R "$third" "$dir/THIRD_PARTY"
	tar -C "$out" -czf "$out/$name.tar.gz" "$name"
	rm -rf "$dir"
done

(cd "$out" && sha256sum ./*.tar.gz | sed 's| \./| |' > SHA256SUMS)
printf "dist:"; for f in "$out"/*; do printf " %s" "${f##*/}"; done; echo
