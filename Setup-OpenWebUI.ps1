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

$scriptPath = Resolve-Path -Path "$PSScriptRoot\update_open-webui.sh"
$wslPath = "/mnt/" + ($scriptPath.Path -replace '\\', '/' -replace ':', '').ToLower()

Start-CommandWithRetry "wsl chmod +x $wslPath"
Write-Log "Running script: ${wslPath}..." -ForegroundColor Cyan
wsl bash -c "$wslPath"

Write-Log "Launching WSL and checking Docker status..." -ForegroundColor Cyan
Start-CommandWithRetry "wsl bash -c 'docker ps'"

Write-Host "Launching WSL interactively with Docker status..." -ForegroundColor Cyan
wsl -e bash -c "docker ps; exec bash"
