Set-StrictMode -Version Latest

if (Get-Module -Name WslCommon) {
    Remove-Module -Name WslCommon -Force
}
Import-Module "$PSScriptRoot\wsl\WslCommon.psm1" -Force

Write-Log "Setting up WSL and Ubuntu..." -ForegroundColor Cyan
Request-AdminPrivileges
Install-WslIfNeeded
Set-WslVersionIfNeeded
Install-WslDistroIfNeeded -DistroName "Ubuntu"

Write-Log "Starting OpenWebUI setup..." -ForegroundColor Cyan
Stop-Wsl

$wslScriptConfigPath = Convert-ToPath -WindowsPath "$PSScriptRoot\update_open-webui.config.sh"
Set-ExecutableAttribute -Path $wslScriptConfigPath

$wslScriptPath = Convert-ToPath -WindowsPath "$PSScriptRoot\update_open-webui.sh"
Set-ExecutableAttribute -Path $wslScriptPath

Start-WslScript -Path $wslScriptPath

Write-Host "Launching WSL interactively with Docker status..." -ForegroundColor Cyan
wsl -e bash -c "docker ps; exec bash"
