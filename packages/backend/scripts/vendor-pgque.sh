#!/usr/bin/env bash
#
# Vendors the PgQue schema installer from upstream at a pinned tag.
#
# PgQue ships no package. It is not on PGXN, it is not a C extension with an OS
# package, and pgque-go is a driver only — it embeds no schema and has no migrate
# command. The DDL that creates the pgque schema exists in exactly one place,
# sql/pgque.sql in the upstream repo, so the only real question is *when* we copy
# it. Copying at deploy time (a clone, or a curl in an entrypoint) makes every
# deploy depend on GitHub being reachable and hides schema changes from review.
# Copying once, here, buys the three properties we actually want: the version is
# pinned, an upgrade shows up as a readable diff of the SQL, and a deploy needs
# nothing but this repo.
#
# third_party/pgque/pgque.sql is not ours and is never hand-edited — the checksum
# exists to make an edit fail loudly. Anything we want on top of it belongs in
# scripts/sql as an ordinary migration that runs after it.
#
# The copy lives in third_party/ and not vendor/ because .gitignore ignores
# vendor/ for Go module vendoring, which would silently leave this untracked.
#
# Usage:
#   ./scripts/vendor-pgque.sh              # re-fetch the pinned version
#   ./scripts/vendor-pgque.sh v0.3.0       # move the pin to a new tag
#   ./scripts/vendor-pgque.sh --verify     # check the working copy against the checksum
#   ./scripts/vendor-pgque.sh --install    # apply it to $DATABASE_URL, then assert the version

set -euo pipefail

readonly REPO="NikolayS/pgque"
readonly SQL_FILE="pgque.sql"
readonly SUM_FILE="pgque.sql.sha256"
readonly VERSION_FILE="VERSION"

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly VENDOR_DIR="${SCRIPT_DIR}/third_party/pgque"

# Matches cmd/config.local.toml. Override with DATABASE_URL for any other target.
readonly DEFAULT_DATABASE_URL="postgres://dev:dev_password_123@localhost:5432/database-dev?sslmode=disable"

die() {
	printf 'error: %s\n' "$*" >&2
	exit 1
}

# usage reprints this file's header comment, so the docs cannot drift from the
# flags below: skip the shebang, strip the leading '# ', stop at the first line
# that is not a comment.
usage() {
	awk 'NR == 1 { next } /^#/ { sub(/^# ?/, ""); print; next } { exit }' "${BASH_SOURCE[0]}"
}

# sha256 papers over the one difference that matters between macOS and CI: BSD
# ships shasum, GNU coreutils ships sha256sum. Both speak the same -c format.
sha256() {
	if command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$@"
	else
		sha256sum "$@"
	fi
}

pinned_version() {
	[[ -f "${VENDOR_DIR}/${VERSION_FILE}" ]] ||
		die "no pin at ${VENDOR_DIR}/${VERSION_FILE} — run '${0##*/} <tag>' to create one"
	tr -d '[:space:]' <"${VENDOR_DIR}/${VERSION_FILE}"
}

# header_version reads the version pgque.sql states about itself, which is the
# only self-description the file carries.
header_version() {
	sed -n 's/^-- Version: *//p' "$1" | head -1 | tr -d '[:space:]'
}

fetch() {
	local version="$1"
	local url="https://raw.githubusercontent.com/${REPO}/${version}/sql/${SQL_FILE}"

	local tmp
	tmp="$(mktemp)"
	# shellcheck disable=SC2064 # expand tmp now, not at trap time
	trap "rm -f '${tmp}'" EXIT

	printf '==> fetching %s@%s\n' "${REPO}" "${version}"
	curl -fsSL "${url}" -o "${tmp}" ||
		die "could not fetch ${url} — does tag ${version} exist?"

	# The tag says what we asked for, the header says what we got. A mismatch
	# means the tag moved or the layout changed upstream, and pinning a checksum
	# over the wrong file is worse than failing here.
	local declared
	declared="$(header_version "${tmp}")"
	[[ "${declared}" == "${version#v}" ]] ||
		die "${url} declares version '${declared}', expected '${version#v}'"

	mkdir -p "${VENDOR_DIR}"
	cp "${tmp}" "${VENDOR_DIR}/${SQL_FILE}"
	chmod 644 "${VENDOR_DIR}/${SQL_FILE}" # mktemp gives 600; this file is committed
	printf '%s\n' "${version}" >"${VENDOR_DIR}/${VERSION_FILE}"
	(cd "${VENDOR_DIR}" && sha256 "${SQL_FILE}" >"${SUM_FILE}")

	printf 'ok: vendored pgque %s (%s lines)\n' \
		"${version}" "$(wc -l <"${VENDOR_DIR}/${SQL_FILE}" | tr -d ' ')"
}

verify() {
	local version
	version="$(pinned_version)"
	[[ -f "${VENDOR_DIR}/${SQL_FILE}" ]] ||
		die "${VENDOR_DIR}/${SQL_FILE} is missing — run '${0##*/}' to restore it"

	(cd "${VENDOR_DIR}" && sha256 -c "${SUM_FILE}" >/dev/null 2>&1) ||
		die "${SQL_FILE} does not match ${SUM_FILE} — it was edited or truncated; run '${0##*/}' to restore the pinned copy"

	# Catches a VERSION file bumped without re-fetching, which the checksum alone
	# cannot see: the SQL is untouched, so it still matches.
	local declared
	declared="$(header_version "${VENDOR_DIR}/${SQL_FILE}")"
	[[ "${declared}" == "${version#v}" ]] ||
		die "${VERSION_FILE} pins ${version} but ${SQL_FILE} is ${declared} — run '${0##*/} ${version}'"

	printf 'ok: pgque %s\n' "${version}"
}

install() {
	verify
	command -v psql >/dev/null 2>&1 || die "psql not found on PATH"

	local version url
	version="$(pinned_version)"
	url="${DATABASE_URL:-${DEFAULT_DATABASE_URL}}"

	# The installer is idempotent by design: re-running it upgrades an existing
	# install in place, so this is both the install and the upgrade path.
	printf '==> installing pgque %s into %s\n' "${version}" "${url##*@}"
	psql "${url}" --no-psqlrc --single-transaction -v ON_ERROR_STOP=1 \
		-f "${VENDOR_DIR}/${SQL_FILE}" >/dev/null

	# What the database reports is the only answer that counts — a file on disk
	# says nothing about which version some other hand installed last.
	local installed
	installed="$(psql "${url}" --no-psqlrc -tAc 'select pgque.version()')"
	[[ "${installed}" == "${version#v}" ]] ||
		die "database reports pgque ${installed}, expected ${version#v}"

	printf 'ok: pgque %s installed\n' "${version}"
}

main() {
	case "${1:-}" in
	-h | --help) usage ;;
	--verify) verify ;;
	--install) install ;;
	"")
		local version
		version="$(pinned_version)"
		fetch "${version}"
		;;
	-*) die "unknown flag: $1" ;;
	*) fetch "$1" ;;
	esac
}

main "$@"
