# Deployment (production)

**Site:** https://inspireonlinebd.com: Cloudflare (proxied) → Nginx on the Contabo VPS (169.58.224.91) → Docker Compose.

## Pipeline

| Workflow | Trigger | What it does |
|---|---|---|
| `.github/workflows/ci.yml` | pull requests, pushes to non-main branches | Gitleaks secret scan, Go fmt/vet/test/build, frontend prettier/lint/build, Docker image builds |
| `.github/workflows/deploy.yml` | push to `main`, manual run | Runs CI, then deploys: uploads code, writes `.env` + JWT keys + origin cert, `docker compose up -d --build`, applies Nginx config, checks health locally and publicly |

The deploy job runs in the GitHub **`production` environment**. Add a required reviewer there if you want a manual approval before each deploy.

## Server layout

- App directory: `/opt/inspire`, owned by the `deploy` user
- Deploy user: `deploy`. Key-only login, member of the `docker` group, password locked.
- Only root command the deploy user can run: `sudo /usr/local/sbin/inspire-apply-nginx /opt/inspire`
- Nginx vhost: `infra/nginx/inspireonlinebd.conf`. It restores the real client IP from Cloudflare and routes:
  - `/` → frontend on port 3000
  - `/v1/` → API on port 8080
  - `/auth/oauth/` → API `/v1/auth/oauth/`
- Firewall (ufw): SSH on 2222, plus 80 and 443
- Ports 3000, 8080 and 8108 are bound to 127.0.0.1 only

## One-time setup

### 1. Bootstrap the server

Log in as root over SSH, then:

```bash
curl -fsSL https://raw.githubusercontent.com/inspireonlineofficial/learning_management_system/main/infra/server-bootstrap.sh -o /root/bootstrap.sh
# or copy infra/server-bootstrap.sh over with scp
bash /root/bootstrap.sh
```

The script prints `SSH_HOST`, `SSH_PORT`, `SSH_USER`, `SSH_KNOWN_HOSTS` and `SSH_PRIVATE_KEY`.

### 2. Create a Cloudflare Origin Certificate

1. Go to Cloudflare → inspireonlinebd.com → SSL/TLS → Origin Server → **Create Certificate**.
2. Keep the defaults (RSA, `*.inspireonlinebd.com` and `inspireonlinebd.com`, 15 years).
3. Copy the certificate into the `CF_ORIGIN_CERT` secret and the private key into `CF_ORIGIN_KEY`.
4. Set SSL/TLS mode to **Full (strict)**.

### 3. Add GitHub secrets

Go to GitHub → Settings → Environments → **New environment** `production` → Add secret.

| Secret | Value |
|---|---|
| `SSH_HOST` | from the bootstrap output |
| `SSH_PORT` | `2222` |
| `SSH_USER` | `deploy` |
| `SSH_KNOWN_HOSTS` | from the bootstrap output |
| `SSH_PRIVATE_KEY` | from the bootstrap output |
| `CF_ORIGIN_CERT` | origin certificate (PEM) |
| `CF_ORIGIN_KEY` | origin private key (PEM) |
| `PROD_ENV` | the complete production `.env` (template below) |

Or use the GitHub CLI: `gh secret set NAME --env production < file`

### `PROD_ENV` template

Generate each random value with `openssl rand -base64 32`. Generate the 32-byte key with `openssl rand -hex 16`.

```dotenv
SERVER_PORT=8080
FRONTEND_BASE_URL=https://inspireonlinebd.com
VITE_API_BASE_URL=https://inspireonlinebd.com

POSTGRES_USER=postgres
POSTGRES_PASSWORD=<random>
POSTGRES_DB=lms
DATABASE_DSN=postgres://postgres:<same password>@postgres:5432/lms?sslmode=disable

REDIS_PASSWORD=<random>
REDIS_URL=redis://:<same password>@redis:6379/0

RUSTFS_ENDPOINT=https://nyc3.digitaloceanspaces.com
RUSTFS_ACCESS_KEY=<new DO Spaces key>
RUSTFS_SECRET_KEY=<new DO Spaces secret>
RUSTFS_REGION=nyc3
RUSTFS_VIDEO_BUCKET=<bucket>
RUSTFS_FILES_BUCKET=<bucket>
RUSTFS_CERTIFICATES_BUCKET=<bucket>
RUSTFS_BOOKS_BUCKET=<bucket>
STORAGE_CORS_ORIGINS=https://inspireonlinebd.com

JWT_ISSUER=lms-backend
JWT_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----"
JWT_PUBLIC_KEY="-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----"

GOOGLE_CLIENT_ID=861493107444-spauveqj5pdv4tpdu8u0f4bm2i8jos78.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=<from Google Cloud>
GOOGLE_REDIRECT_URL=https://inspireonlinebd.com/auth/oauth/google/callback
OAUTH_TOKEN_ENCRYPTION_KEY=<32-byte random>

TYPESENSE_HOST=typesense
TYPESENSE_PORT=8108
TYPESENSE_API_KEY=<random>

SMTP_HOST=<smtp relay>
SMTP_PORT=587
SMTP_TLS=starttls
SMTP_USERNAME=<user>
SMTP_PASSWORD=<password>
SMTP_FROM=no-reply@inspireonlinebd.com
SMTP_FROM_NAME=Inspire Online

ADMIN_DEV_BYPASS=false
```

To generate the JWT key pair:

```bash
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out private.pem
openssl pkey -in private.pem -pubout -out public.pem
```

## Day-to-day

- **Deploy:** merge or push to `main`. To deploy without a code change, go to Actions → Deploy → **Run workflow**.
- **Change a setting:** edit the `PROD_ENV` secret, then re-run Deploy.
- **Logs:** `ssh -p 2222 deploy@169.58.224.91 'cd /opt/inspire && docker compose logs -f app'`
- **Roll back:** revert the commit on `main`. The push redeploys the previous code.
