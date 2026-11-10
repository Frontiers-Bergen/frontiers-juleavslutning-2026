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
- **The app listens on 127.0.0.1 only** (`HOST` in the service file): Caddy, with TLS, is the only way in. Without it, `<server IP>:<port>` answered over plain HTTP from the internet; the server's firewall does not block these ports.
- **NATS keeps nothing and opens no port**: it runs inside the app's process, with core messages only (no JetStream, no store directory). Anything that must survive a restart is in SQLite.
- **main keeps its database across deploys**; a preview starts empty and its database is deleted with the preview.
- **Schema changes on main** need the database deleted by hand (there are no migrations): `sudo systemctl stop juleavslutning-main`, remove `/opt/juleavslutning/main/presentation.db*`, start again. Never on the evening itself.
- **Previews only for pull requests from this repository**, not forks: GitHub gives fork PRs no secrets. Workshop attendees who want a preview get write access; see "Collaborators and trust" below.

## Collaborators and trust

Workshop attendees who want to push are added as collaborators with write access, and removed when the project ends:

```sh
gh api -X PUT repos/frontiers-bergen/frontiers-juleavslutning-2026/collaborators/<username> -f permission=push
gh api -X DELETE repos/frontiers-bergen/frontiers-juleavslutning-2026/collaborators/<username>
```

Write access is more than it looks: a preview runs the pull request's own code on the shared server (the app as the
`deploy` user, and the pull request's `deploy/deploy.sh` as root through sudo), and `main` is not protected.
We decided (7 November 2026) not to add deploy approval or branch protection:
- It is a social question, not a technical one: there are many ways to spoil an evening for everyone, and we trust the people in the room.
- The attack surface is small: Regncon 2026 is over, the 2027 site is not in production yet, and the server holds
  little or no personal information.
- Approval steps would slow down the workshop, where a merged change should reach the big screen in minutes.
- Everything is in git, and the code is reviewed before it goes live on the juleavslutning.

Fork pull requests run the checks without secrets and get no preview.

## One-time setup

On the server, as root. The `deploy` user already exists for conorganizer.

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

A dedicated SSH key for this project, so it can be revoked on its own when the project ends. On a laptop with `gh`:

```sh
ssh-keygen -t ed25519 -N "" -C "github-actions deploy juleavslutning" -f ~/.ssh/juleavslutning-deploy
gh secret set HETZNER_SSH_KEY --env Production --repo frontiers-bergen/frontiers-juleavslutning-2026 < ~/.ssh/juleavslutning-deploy
scp ~/.ssh/juleavslutning-deploy.pub gerhard@regncon.no:
```

On the server (the deploy user's home is `/opt/conorganizer`):

```sh
sudo tee -a /opt/conorganizer/.ssh/authorized_keys < ~/juleavslutning-deploy.pub
rm ~/juleavslutning-deploy.pub
```

In the GitHub repository (Settings → Secrets and variables → Actions):
- Secret `HETZNER_SSH_KEY` in the **Production** environment (above). The deploy jobs use `environment: Production`.
- Repository variables (not secrets): `HETZNER_HOST` = `regncon.no` (it has an IPv4 address; GitHub's runners have no IPv6),
  `HETZNER_USER` = `deploy`, and `DEPLOY_ENABLED` = `true`. Until `DEPLOY_ENABLED` is set, CI only runs the checks.
  It must be a repository variable: environment variables are not available when GitHub decides whether a job runs.

Every step runs on the server in a root shell (`sudo -i`), or with `sudo tee` instead of `>` redirects: a redirect is
done by your own shell, before `sudo`, and fails in `/etc`.

## Security note

As with conorganizer, the deploy key can upload a `deploy.sh` and run it as root, so the key is as powerful as root
on the server. Keep it in GitHub secrets only.

## When the project ends

Remove the `github-actions deploy juleavslutning` line from `/opt/conorganizer/.ssh/authorized_keys`, `/etc/sudoers.d/deploy-juleavslutning`, `/etc/juleavslutning`, `/opt/juleavslutning`,
`/etc/systemd/system/juleavslutning-*.service` and `/etc/caddy/sites-enabled/juleavslutning-*.caddy`,
then `systemctl daemon-reload && systemctl reload caddy`. Back up `/opt/juleavslutning/main/presentation.db` first:
the ideas and feedback from the juleavslutning are analysed from it.
