#!/usr/bin/env bash
# One-time production server setup for Inspire LMS (Ubuntu 24.04).
# Run as root ON the server:  bash infra/server-bootstrap.sh
#
# What it does:
#   - installs Docker + Compose, Nginx, ufw
#   - creates a non-root "deploy" user (docker group, key-only login)
#   - installs a root-owned helper that applies the Nginx config (the only sudo
#     command the deploy user may run)
#   - generates the CI deploy SSH key and prints the GitHub secret values
#   - firewall: allows SSH (port below), 80, 443
set -euo pipefail

SSH_PORT="${SSH_PORT:-2222}"
DEPLOY_USER="deploy"
DEPLOY_PATH="/opt/inspire"
PUBLIC_IP="$(curl -4 -fsS https://api.ipify.org || hostname -I | awk '{print $1}')"

[ "$(id -u)" -eq 0 ] || { echo "run as root"; exit 1; }

echo "==> Packages"
apt-get update -y
DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl nginx ufw
if ! command -v docker >/dev/null; then
  curl -fsSL https://get.docker.com | sh
fi
systemctl enable --now docker nginx

echo "==> Deploy user"
id "$DEPLOY_USER" >/dev/null 2>&1 || useradd -m -s /bin/bash "$DEPLOY_USER"
usermod -aG docker "$DEPLOY_USER"
passwd -l "$DEPLOY_USER" >/dev/null           # no password login, key only
install -d -o "$DEPLOY_USER" -g "$DEPLOY_USER" -m 750 "$DEPLOY_PATH"

echo "==> CI deploy key"
KEYDIR="$(mktemp -d)"
ssh-keygen -q -t ed25519 -N "" -C "github-actions-deploy@inspireonlinebd" -f "$KEYDIR/id"
install -d -o "$DEPLOY_USER" -g "$DEPLOY_USER" -m 700 "/home/$DEPLOY_USER/.ssh"
cat "$KEYDIR/id.pub" > "/home/$DEPLOY_USER/.ssh/authorized_keys"
chown "$DEPLOY_USER:$DEPLOY_USER" "/home/$DEPLOY_USER/.ssh/authorized_keys"
chmod 600 "/home/$DEPLOY_USER/.ssh/authorized_keys"

echo "==> Nginx apply helper"
cat > /usr/local/sbin/inspire-apply-nginx <<'HELPER'
#!/usr/bin/env bash
# Installs the repo's Nginx vhost + Cloudflare origin cert, then reloads Nginx.
set -euo pipefail
SRC="${1:-/opt/inspire}"
case "$SRC" in /opt/inspire) ;; *) echo "refusing path $SRC"; exit 1;; esac
install -d -m 755 /etc/ssl/cloudflare
install -m 644 "$SRC/tls/inspireonlinebd.pem" /etc/ssl/cloudflare/inspireonlinebd.pem
install -m 600 "$SRC/tls/inspireonlinebd.key" /etc/ssl/cloudflare/inspireonlinebd.key
install -m 644 "$SRC/infra/nginx/inspireonlinebd.conf" /etc/nginx/sites-available/inspireonlinebd.conf
ln -sfn /etc/nginx/sites-available/inspireonlinebd.conf /etc/nginx/sites-enabled/inspireonlinebd.conf
rm -f /etc/nginx/sites-enabled/default /etc/nginx/sites-enabled/inspire /etc/nginx/sites-enabled/inspire-lms.conf
nginx -t
systemctl reload nginx
HELPER
chmod 755 /usr/local/sbin/inspire-apply-nginx
echo "$DEPLOY_USER ALL=(root) NOPASSWD: /usr/local/sbin/inspire-apply-nginx" > /etc/sudoers.d/inspire-deploy
chmod 440 /etc/sudoers.d/inspire-deploy
visudo -cf /etc/sudoers.d/inspire-deploy

echo "==> Firewall"
ufw allow "$SSH_PORT"/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

KNOWN_HOSTS="$(ssh-keyscan -p "$SSH_PORT" -t ed25519,ecdsa,rsa 127.0.0.1 2>/dev/null | sed "s/^\[127.0.0.1\]:$SSH_PORT/[$PUBLIC_IP]:$SSH_PORT/")"

cat <<EOF

=====================================================================
 Server ready. Add these as GitHub secrets (environment: production):

 SSH_HOST          = $PUBLIC_IP
 SSH_PORT          = $SSH_PORT
 SSH_USER          = $DEPLOY_USER

 SSH_KNOWN_HOSTS   = (everything between the lines)
---------------------------------------------------------------------
$KNOWN_HOSTS
---------------------------------------------------------------------

 SSH_PRIVATE_KEY   = (everything between the lines, including BEGIN/END)
---------------------------------------------------------------------
$(cat "$KEYDIR/id")
---------------------------------------------------------------------
 Copy it now — it is deleted from this server when the script ends.
=====================================================================
EOF
shred -u "$KEYDIR/id" 2>/dev/null || rm -f "$KEYDIR/id"
rm -rf "$KEYDIR"
