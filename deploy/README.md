# Setting up a chatgamelab box

One instance runs on one box: Debian 13 or newer, a run user `cgl` holding
rootless podman and the app's stack, native nginx in front with a Let's
Encrypt certificate. The box builds nothing; it installs released versions.

## An instance is one file

An instance's configuration is one YAML file named after its domain, e.g.
`chatgamelab.example.org.yml`. `example.yml` lists every key. The file holds
public values only (Auth0 identifiers, Sentry DSNs, the backup target, the SSH
keys allowed in), so where it lives is up to its operator.

```
deploy/setup.sh path/to/chatgamelab.example.org.yml --admin you@example.org --backup-key ~/.ssh/cgl-backup
```

`setup.sh` runs ansible against the domain as root and is safe to run again:
each run brings the box back to its file. It needs ansible and ssh on the
operator's machine, root SSH access to the box (a key installed through the
provider's console) and DNS for the domain pointing at the box.

## What a box holds

| Where | What |
|---|---|
| `cgl` (`/var/lib/cgl`) | rootless podman, its socket, the stack and the database volume |
| `~cgl/.chatgamelab/etc/env` | the instance's values from its file, plus the secrets kept on the box: `DB_PASS` (generated at the first run) and `ADMIN_EMAILS` (from `--admin`) |
| `~cgl/.chatgamelab/etc/backup_key` | the SFTP backup key (from `--backup-key`) |
| nginx | the one public entry point: TLS, proxying to the app's port 7100 on localhost |
| `chatgamelab-boot.service` (user unit of `cgl`) | starts the stack's containers after a reboot |

An instance can put a shared password in front of the whole site, e.g. a
development instance: `basic_auth_user` in its file, the password given with
`--basic-auth` (asked for) or `--basic-auth-file` (read from a file, e.g. a
keychain), kept on the box as a hash in `/etc/nginx/htpasswd-<domain>`; the
same password again leaves the hash as it is. `/api/health` and certificate renewal stay
open. An empty `basic_auth_user` removes the protection.

A provider that assigns an IPv6 address without announcing it gets
`ipv6_address` in the instance file; `setup.sh` adds it to the box's netplan
configuration.

The first admin is `admin_email` in the instance file, or the address given
with `--admin`: it becomes admin at its first sign-in, while the instance has
no admin.

Backups are schemaf's built-in SFTP backup, nightly at `backup_hour`. An empty
`backup_ssh_host` switches them off.

## Versions

Each chatgamelab release carries a signed artifact. The box fetches the
releases on its `release_channel`, verifies each one against the release
workflow of `release_repo` and offers the newer ones to the instance's admins
in the app, who start an upgrade with a click. An upgrade backs up the
database first and rolls back when the new version fails its health check.

`setup.sh` runs from a checkout of a release tag (`git checkout v<version>`);
the box's run-side `schemaf.sh` comes from that tag. On a new box it installs
the newest release on the channel and enables schemaf's updater, which checks
for releases daily, and at once when an admin clicks "Check for updates".

By hand, as `cgl`:

```
~/.chatgamelab/bin/schemaf.sh releases            # fetch and verify the newer releases
~/.chatgamelab/bin/schemaf.sh install <version>   # what the admin's click does
~/.chatgamelab/bin/schemaf.sh updater status
```

## A box that ran chatgamelab 1.x

`setup.sh` converts it. First it compares the 1.x settings with the instance
file: when 1.x reports errors to Sentry or backs up, and the file leaves those
values empty, the run stops before changing anything and prints the lines to
add. Then it removes the 1.x stack with its database, its units, the build user
`cgl-build` and the 1.x files, keeps the SFTP backup key, and sets the box up
as above.
