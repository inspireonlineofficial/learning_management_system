# One-time: install the new CI/CD workflows, scrub leaked secrets from history,
# and force-push main. Run from the repo root in PowerShell:
#     powershell -ExecutionPolicy Bypass -File infra\ci\apply-cicd.ps1
$ErrorActionPreference = "Stop"
Set-Location (git rev-parse --show-toplevel)

if (-not (Test-Path ".env.scrub")) { throw ".env.scrub not found in repo root" }

Write-Host "==> Backing up current history (local bundle)" -ForegroundColor Cyan
$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$backup = Join-Path (Split-Path (Get-Location) -Parent) "lms-history-backup-$stamp.bundle"
git bundle create $backup --all
Write-Host "    saved $backup"

Write-Host "==> Installing workflows" -ForegroundColor Cyan
New-Item -ItemType Directory -Force .github\workflows | Out-Null
Get-ChildItem .github\workflows -Filter *.yml | ForEach-Object { git rm -q --cached $_.FullName 2>$null; Remove-Item $_.FullName }
Copy-Item infra\ci\ci.yml     .github\workflows\ci.yml
Copy-Item infra\ci\deploy.yml .github\workflows\deploy.yml

Write-Host "==> Removing old-domain Nginx configs and tracked junk" -ForegroundColor Cyan
git rm -q --ignore-unmatch infra/nginx/inspire-online.conf infra/nginx/inspire-online-http.conf
git rm -r -q --cached --ignore-unmatch .docker_tmp
git rm -q --cached --ignore-unmatch infra/ci/ci.yml infra/ci/deploy.yml
Remove-Item infra\ci\ci.yml, infra\ci\deploy.yml -ErrorAction SilentlyContinue

git add .github/workflows infra docker-compose.yml .gitignore DEPLOYMENT.md
git commit -m "ci: rebuild CI/CD for inspireonlinebd.com (deploy user, Cloudflare origin TLS, OAuth route)"

Write-Host "==> Scrubbing leaked secrets from all history" -ForegroundColor Cyan
python -m pip install --quiet --user git-filter-repo
$remote = git remote get-url origin
python -m git_filter_repo --replace-text .env.scrub --force
git remote add origin $remote

Write-Host "==> Verifying" -ForegroundColor Cyan
foreach ($line in Get-Content .env.scrub) {
  $value = ($line -split "==>")[0]
  $hits = git log --all -p -S $value --format=%h
  if ($hits) { throw "leaked value still present in history" }
}
Remove-Item .env.scrub
Write-Host "    history clean; .env.scrub deleted"

Write-Host "==> Force-pushing main" -ForegroundColor Cyan
git push --force origin main
git push --force --tags origin

Write-Host ""
Write-Host "Done. Pushing main started the Deploy workflow; it will stop at"
Write-Host "'Check required secrets' until you add the GitHub secrets (see DEPLOYMENT.md)."
Write-Host "Backup of the old history: $backup  (delete it once everything works)"
