$wshShell = New-Object -ComObject WScript.Shell
$startupPath = [System.Environment]::GetFolderPath('Startup')
$shortcutPath = Join-Path $startupPath "VPN Guardian.lnk"

$exePath = "C:\Users\bezukladnikovma\.gemini\antigravity\scratch\vpn-guardian\vpn-guardian.exe"

$shortcut = $wshShell.CreateShortcut($shortcutPath)
$shortcut.TargetPath = $exePath
$shortcut.WorkingDirectory = "C:\Users\bezukladnikovma\.gemini\antigravity\scratch\vpn-guardian"
$shortcut.Description = "VPN Guardian - Автоконтроль корпоративного подключения SG"
$shortcut.Save()

Write-Host "Ярлык успешно добавлен в автозагрузку: $shortcutPath" -ForegroundColor Green
