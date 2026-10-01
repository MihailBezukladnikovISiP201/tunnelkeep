$wshShell = New-Object -ComObject WScript.Shell
$startupPath = [System.Environment]::GetFolderPath('Startup')
$shortcutPath = Join-Path $startupPath "TunnelKeep.lnk"

# Remove legacy shortcut if existed
$oldShortcut = Join-Path $startupPath "VPN Guardian.lnk"
if (Test-Path $oldShortcut) { Remove-Item -Force $oldShortcut }

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$exePath = Join-Path $scriptDir "tunnelkeep.exe"

$shortcut = $wshShell.CreateShortcut($shortcutPath)
$shortcut.TargetPath = $exePath
$shortcut.WorkingDirectory = $scriptDir
$shortcut.Description = "TunnelKeep - Автоконтроль корпоративного VPN туннеля"
$shortcut.Save()

Write-Host "Ярлык успешно добавлен в автозагрузку: $shortcutPath" -ForegroundColor Green
