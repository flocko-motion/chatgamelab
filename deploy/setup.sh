#!/usr/bin/env bash
# Sets up a box for one chatgamelab instance, or brings it back to its
# configuration: packages, the run user `cgl`, rootless podman, nginx with TLS,
# and the instance's env file. Safe to run again.
#
#   deploy/setup.sh <dir>/<domain>.yml [--admin <email>] [--backup-key <file>]
#
# <domain>.yml   the instance's configuration (see deploy/example.yml); its
#                file name is the domain the box serves and SSH reaches.
# --admin        the address that becomes admin at its first sign-in, while no
#                admin exists (ADMIN_EMAILS). Kept on the box.
# --backup-key   the private SSH key for the SFTP backup target. Copied to
#                the box.
#
# Needs ansible and ssh on this machine, and root SSH access to the box.
# Written for bash 3.2 (macOS).
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"

usage() {
  sed -n '6,13p' "$0" | sed 's/^# \{0,1\}//' >&2
  exit 2
}

config=""
admin=""
backup_key=""
while [ $# -gt 0 ]; do
  case "$1" in
    --admin) [ $# -ge 2 ] || usage; admin="$2"; shift 2 ;;
    --backup-key) [ $# -ge 2 ] || usage; backup_key="$2"; shift 2 ;;
    -h|--help) usage ;;
    -*) echo "unknown option: $1" >&2; usage ;;
    *) [ -z "$config" ] || usage; config="$1"; shift ;;
  esac
done
[ -n "$config" ] || usage

for tool in ansible-playbook ansible-galaxy ssh; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "$tool is missing. Install ansible and openssh: apt install ansible openssh-client | brew install ansible" >&2
    exit 1
  fi
done

[ -f "$config" ] || { echo "no such file: $config" >&2; exit 1; }
config="$(cd "$(dirname "$config")" && pwd)/$(basename "$config")"
domain="$(basename "$config" .yml)"
case "$domain" in
  *.*) ;;
  *) echo "the file name names the domain, e.g. chatgamelab.example.org.yml; got $(basename "$config")" >&2; exit 1 ;;
esac

if [ -n "$backup_key" ]; then
  [ -f "$backup_key" ] || { echo "no such file: $backup_key" >&2; exit 1; }
  backup_key="$(cd "$(dirname "$backup_key")" && pwd)/$(basename "$backup_key")"
fi

# The box's run-side schemaf.sh comes from the release this checkout is at.
setup_ref="$(git -C "$here" describe --tags --exact-match 2>/dev/null || true)"
case "$setup_ref" in
  v[0-9]*) ;;
  *) echo "check out a release tag first (git checkout v<version>): the box's schemaf.sh comes from it" >&2; exit 1 ;;
esac

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
chmod 700 "$work"

printf '[chatgamelab]\n%s ansible_user=root\n' "$domain" > "$work/inventory.ini"

# The admin address and the key path travel in a file rather than on the
# command line, where every process on this machine could read them.
{
  printf 'admin_email: "%s"\n' "$admin"
  printf 'backup_key_src: "%s"\n' "$backup_key"
  printf 'setup_ref: "%s"\n' "$setup_ref"
} > "$work/given.yml"

cd "$here"
ansible-galaxy collection install -r requirements.yml >/dev/null
ansible-playbook \
  -i "$work/inventory.ini" \
  -e "@$config" \
  -e "@$work/given.yml" \
  playbook.yml

echo "✓ $domain is set up."
