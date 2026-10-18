#!/bin/sh
set -eu

fail() {
	printf 'tap: %s\n' "$*" >&2
	exit 1
}

download() {
	curl -fsSL --proto '=https' --proto-redir '=https' --retry 3 --connect-timeout 30 "$@"
}

as_root() {
	if [ "$user_id" = 0 ]; then
		"$@"
	else
		sudo -- "$@"
	fi
}

cleanup() {
	if [ -n "$staging" ]; then
		as_root rm -rf -- "$staging"
	fi

	if [ -n "$temporary" ]; then
		rm -rf -- "$temporary"
	fi
}

main() {
	[ "$(uname -s)" = Linux ] || fail 'this installer requires Linux.'
	[ "$(uname -m)" = x86_64 ] || fail 'only Linux amd64 (x86_64) is supported.'

	for dependency in curl sha256sum mktemp install mv; do
		command -v "$dependency" >/dev/null 2>&1 || fail "required command not found: $dependency"
	done

	user_id=$(id -u)
	if [ "$user_id" != 0 ]; then
		command -v sudo >/dev/null 2>&1 || fail 'installing to /usr/local/bin requires root or sudo.'
	fi

	temporary=''
	staging=''
	trap cleanup 0
	trap 'exit 130' INT
	trap 'exit 143' TERM
	temporary=$(mktemp -d)

	repository=https://github.com/coalaura/tap
	release_url=$(download -o /dev/null -w '%{url_effective}' "$repository/releases/latest")
	case "$release_url" in
		"$repository/releases/tag/"*) tag=${release_url##*/} ;;
		*) fail 'could not resolve the latest stable release.' ;;
	esac

	case "$tag" in
		''|*[!A-Za-z0-9.+-]*) fail 'invalid release tag.' ;;
	esac

	asset=tap-linux-amd64
	printf 'Downloading tap %s...\n' "$tag"
	download -o "$temporary/$asset" "$repository/releases/download/$tag/$asset"
	download -o "$temporary/SHA256SUMS" "$repository/releases/download/$tag/SHA256SUMS"

	checksum=''
	while read -r digest filename extra; do
		if [ "$filename" = "$asset" ] || [ "$filename" = "*$asset" ]; then
			[ -z "$checksum" ] || fail 'duplicate release checksum.'
			checksum=$digest
		fi
	done < "$temporary/SHA256SUMS"

	[ "${#checksum}" = 64 ] || fail 'missing or invalid release checksum.'
	case "$checksum" in
		*[!A-Fa-f0-9]*) fail 'invalid release checksum.' ;;
	esac

	printf '%s  %s\n' "$checksum" "$temporary/$asset" | sha256sum -c - || fail 'release checksum verification failed.'

	as_root install -d -m 755 /usr/local/bin
	# Stage on the destination filesystem so replacement is atomic, even while tap is running.
	staging=$(as_root mktemp -d /usr/local/bin/.tap-install-XXXXXX)
	as_root install -m 755 -o 0 -g 0 "$temporary/$asset" "$staging/tap"
	as_root mv -fT -- "$staging/tap" /usr/local/bin/tap

	printf 'Installed tap %s to /usr/local/bin/tap\n' "$tag"
	case ":${PATH:-}:" in
		*:/usr/local/bin:*) ;;
		*) printf 'Add /usr/local/bin to your shell PATH: export PATH="/usr/local/bin:$PATH"\n' ;;
	esac
}

main "$@"
