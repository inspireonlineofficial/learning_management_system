# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Inspire LMS, please **do not** open a public GitHub issue.

Instead, report it privately to the repository maintainers (for example via GitHub Security Advisories or by contacting `contact@inspire-online.com`) with:

- A clear description of the vulnerability and its potential impact.
- Steps to reproduce or a minimal proof of concept.
- Any affected endpoints, routes, or configuration details.

We will acknowledge receipt and work to investigate and remediate confirmed issues promptly.

## Security Practices in This Repository

- **Secret Scanning**: Every push and pull request is scanned by Gitleaks in CI.
- **Authentication**: Passwords are hashed with bcrypt; tokens are signed using RS256 asymmetric keys with refresh token revocation in Redis.
- **Authorization**: Protected endpoints enforce JWT validation and role checks (`student`, `teacher`, `admin`) with RBAC permissions and audit logging.
- **Input & Content Safety**: Forum Markdown is sanitized with Bluemonday to mitigate XSS; file uploads validate size, MIME type, and magic bytes; object storage keys are never exposed publicly (presigned URLs only).
- **Container Hardening**: Docker containers run as non-root users (`appuser` / `node`) with `no-new-privileges:true` and read-only root filesystems where supported.
