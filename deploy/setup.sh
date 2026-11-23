#!/usr/bin/env bash
# Sets up a box for one chatgamelab instance, or brings it back to its
# configuration: packages, the run user `cgl`, rootless podman, nginx with TLS,
# and the instance's env file. Safe to run again.
#
#   deploy/setup.sh <dir>/<domain>.yml [--admin <email>] [--backup-key <file>]
#                   [--basic-auth | --basic-auth-file <file>]
#
# <domain>.yml   the instance's configuration (see deploy/example.yml); its
#                file name is the domain the box serves and SSH reaches.
# --admin        the address that becomes admin at its first sign-in, while no
#                admin exists (ADMIN_EMAILS); overrides admin_email in the file.
# --backup-key   the private SSH key for the SFTP backup target. Copied to
#                the box.
# --basic-auth   ask for the password of basic_auth_user (the instance file's
#                shared password in front of the site); its hash goes to the box.
# --basic-auth-file  the same password, read from a file (e.g. from a keychain).
#
# Needs ansible and ssh on this machine, and root SSH access to the box.
# Written for bash 3.2 (macOS).
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"

usage() {
  sed -n '6,17p' "$0" | sed 's/^# \{0,1\}//' >&2
  exit 2
}

config=""
admin=""
backup_key=""
basic_auth=""
basic_auth_file=""
while [ $# -gt 0 ]; do
  case "$1" in
    --admin) [ $# -ge 2 ] || usage; admin="$2"; shift 2 ;;
    --backup-key) [ $# -ge 2 ] || usage; backup_key="$2"; shift 2 ;;
    --basic-auth) basic_auth=1; shift ;;
    --basic-auth-file)
      [ $# -ge 2 ] || usage
      [ -f "$2" ] || { echo "no such file: $2" >&2; exit 1; }
      basic_auth_file="$2"; shift 2 ;;
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

basic_auth_password=""
if [ -n "$basic_auth_file" ]; then
  basic_auth_password="$(cat "$basic_auth_file")"
  [ -n "$basic_auth_password" ] || { echo "$basic_auth_file is empty" >&2; exit 1; }
elif [ -n "$basic_auth" ]; then
  printf 'Password for the site (basic_auth_user): ' >&2
  read -rs basic_auth_password; echo >&2
  printf 'Again: ' >&2
  read -rs again; echo >&2
  [ "$basic_auth_password" = "$again" ] || { echo "the two passwords differ" >&2; exit 1; }
  [ -n "$basic_auth_password" ] || { echo "an empty password protects nothing" >&2; exit 1; }
fi

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
chmod 700 "$work"

printf '[chatgamelab]\n%s ansible_user=root ansible_python_interpreter=/usr/bin/python3\n' "$domain" > "$work/inventory.ini"

# What the flags give travels in a file rather than on the command line,
# where every process on this machine could read it. A flag left out leaves
# the instance file's value standing (admin_email).
yaml_str() { printf '%s' "$1" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))'; }
{
  printf 'setup_ref: %s\n' "$(yaml_str "$setup_ref")"
  [ -z "$admin" ] || printf 'admin_email: %s\n' "$(yaml_str "$admin")"
  [ -z "$backup_key" ] || printf 'backup_key_src: %s\n' "$(yaml_str "$backup_key")"
  [ -z "$basic_auth_password" ] || printf 'basic_auth_password: %s\n' "$(yaml_str "$basic_auth_password")"
} > "$work/given.yml"

cd "$here"
ansible-galaxy collection install -r requirements.yml >/dev/null
ansible-playbook \
  -i "$work/inventory.ini" \
  -e "@$config" \
  -e "@$work/given.yml" \
  playbook.yml

echo "✓ $domain is set up."
