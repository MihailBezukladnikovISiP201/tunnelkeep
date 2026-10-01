$startupPath = [System.Environment]::GetFolderPath('Startup')
$shortcutPath = Join-Path $startupPath "TunnelKeep.lnk"
$oldShortcut = Join-Path $startupPath "VPN Guardian.lnk"

if (Test-Path $oldShortcut) { Remove-Item -Force $oldShortcut }

if (Test-Path $shortcutPath) {
    Remove-Item -Force $shortcutPath
    Write-Host "Ярлык удален из автозагрузки: $shortcutPath" -ForegroundColor Green
} else {
    Write-Host "Ярлык автозагрузки не найден." -ForegroundColor Yellow
}
