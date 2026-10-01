param(
    [Parameter(Mandatory=$false)]
    [string]$RepoUrl = "https://github.com/MihailBezukladnikovISiP201/tunnelkeep.git"
)

Write-Host "=== Публикация TunnelKeep на GitHub ===" -ForegroundColor Cyan

if (-not (Test-Path ".git")) {
    git init -b main
    Write-Host "Git репозиторий инициализирован." -ForegroundColor Green
}

git add .
git commit -m "feat: release TunnelKeep v1.0.0"

# Create release tag
git tag -f -a "v1.0.0" -m "Release v1.0.0: TunnelKeep with System Tray and Multi-Adapter support"

if ($RepoUrl -ne "") {
    git remote remove origin 2>$null
    git remote add origin $RepoUrl
    Write-Host "Отправка кода и тегов в $RepoUrl..." -ForegroundColor Yellow
    git push -u origin main --tags --force
    Write-Host "✅ Успешно опубликовано в $RepoUrl!" -ForegroundColor Green
}
