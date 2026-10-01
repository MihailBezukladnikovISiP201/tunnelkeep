param(
    [Parameter(Mandatory=$false)]
    [string]$RepoUrl = ""
)

Write-Host "=== Публикация VPN Guardian на GitHub ===" -ForegroundColor Cyan

if (-not (Test-Path ".git")) {
    git init -b main
    Write-Host "Git репозиторий инициализирован." -ForegroundColor Green
}

git add .
git commit -m "feat: initial release v1.0.0 of VPN Guardian"

# Create release tag
git tag -f -a "v1.0.0" -m "Release v1.0.0: VPN Guardian with System Tray and Multi-Adapter support"

if ($RepoUrl -eq "") {
    $currentRemote = git remote get-url origin 2>$null
    if ($currentRemote) {
        $RepoUrl = $currentRemote
    } else {
        $RepoUrl = Read-Host "Введите URL вашего нового репозитория на GitHub (например, https://github.com/username/vpn-guardian.git)"
    }
}

if ($RepoUrl -ne "") {
    git remote remove origin 2>$null
    git remote add origin $RepoUrl
    Write-Host "Отправка кода и тегов в $RepoUrl..." -ForegroundColor Yellow
    git push -u origin main --tags --force
    Write-Host "✅ Успешно опубликовано! GitHub Actions автоматически запустит сборку релизов." -ForegroundColor Green
} else {
    Write-Host "URL репозитория не указан. Локальный коммит и тег v1.0.0 созданы." -ForegroundColor Yellow
}
