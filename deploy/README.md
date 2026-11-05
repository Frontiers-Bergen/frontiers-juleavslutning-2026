# Deploy

The app runs on the lekeplassen server, next to conorganizer, until the project ends after the juleavslutning.
`.github/workflows/ci-cd.yml` deploys it; the files in this folder are what ends up on the server.

| Environment | Address | Port | Directory |
|---|---|---|---|
| Big screen (`main`) | https://juleavslutning.lekeplassen.regncon.no | 23000 | `/opt/juleavslutning/main` |
| Pull request preview | https://juleavslutning-&lt;PR&gt;.lekeplassen.regncon.no | 23000 + PR | `/opt/juleavslutning/pr-<PR>` |

Choices, and why:
- **Ports 23000 and up, and the `juleavslutning` name**, so nothing collides with conorganizer (19080–19082, 20000 + PR, `<PR>-merge.lekeplassen…`).
- **One subdomain level** (`juleavslutning-42`, not `42.juleavslutning`): the existing `*.lekeplassen.regncon.no` wildcard DNS covers only one level. No DNS changes are needed.
- **Nothing on the Hetzner volume.** Each environment's binary, static files and `presentation.db` live on the server's own disk in `/opt/juleavslutning/<name>`.
- **NATS keeps nothing**: it uses the service's systemd runtime directory (`/run/juleavslutning-<name>`), in memory and wiped on restart. Anything that must survive a restart is in SQLite.
- **main keeps its database across deploys**; a preview starts empty and its database is deleted with the preview.
- **Schema changes on main** need the database deleted by hand (there are no migrations): `sudo systemctl stop juleavslutning-main`, remove `/opt/juleavslutning/main/presentation.db*`, start again. Never on the evening itself.
- **Previews only for pull requests from this repository**, not forks: GitHub gives fork PRs no secrets. Workshop attendees need push access to the repository to get a preview.

## One-time setup

On the server, as root. The `deploy` user and its SSH key already exist for conorganizer.

```sh
# 1. Let the deploy user run our deploy and cleanup scripts as root (mv, chown, chmod and systemctl
#    are already allowed in /etc/sudoers.d/deploy-conorganizer).
cat > /etc/sudoers.d/deploy-juleavslutning <<'SUDOERS'
deploy ALL=(root) NOPASSWD: /opt/juleavslutning/*/deploy.sh, /opt/juleavslutning/*/cleanup.sh
SUDOERS
chmod 440 /etc/sudoers.d/deploy-juleavslutning
visudo -cf /etc/sudoers.d/deploy-juleavslutning

# 2. A directory the deploy user can upload to.
mkdir -p /opt/juleavslutning
chown deploy:www-data /opt/juleavslutning

# 3. The secrets, shared by main and the previews. Generate new values; never reuse ones from git history.
mkdir -p /etc/juleavslutning
cat > /etc/juleavslutning/env <<ENV
INVITE_KEY=$(openssl rand -hex 8)
PRESENTER_KEY=$(openssl rand -hex 8)
SESSION_SECRET=$(openssl rand -hex 32)
ENV
chmod 600 /etc/juleavslutning/env
cat /etc/juleavslutning/env   # note the invite and presenter keys
```

In the GitHub repository (Settings → Secrets and variables → Actions):
- Secrets `HETZNER_HOST`, `HETZNER_USER` (`deploy`) and `HETZNER_SSH_KEY` (the deploy user's private key, the same as in conorganizer).
- Variable `DEPLOY_ENABLED` = `true`. Until it is set, CI only runs the checks.

## Security note

As with conorganizer, the deploy key can upload a `deploy.sh` and run it as root, so the key is as powerful as root
on the server. Keep it in GitHub secrets only.

## When the project ends

Remove `/etc/sudoers.d/deploy-juleavslutning`, `/etc/juleavslutning`, `/opt/juleavslutning`,
`/etc/systemd/system/juleavslutning-*.service` and `/etc/caddy/sites-enabled/juleavslutning-*.caddy`,
then `systemctl daemon-reload && systemctl reload caddy`. Back up `/opt/juleavslutning/main/presentation.db` first:
the ideas and feedback from the juleavslutning are analysed from it.
