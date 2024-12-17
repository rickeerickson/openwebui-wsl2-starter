Set-StrictMode -Version Latest

if (Get-Module -Name WslCommon) {
    Remove-Module -Name WslCommon -Force
}
Import-Module "$PSScriptRoot\wsl\WslCommon.psm1" -Force

Write-Log "Installing WSL and Ubuntu..." -ForegroundColor Cyan
Request-AdminPrivileges
Install-WslIfNeeded
Set-WslVersionIfNeeded
Install-WslDistroIfNeeded -DistroName "Ubuntu"

Write-Log "WSL and Ubuntu installation complete." -ForegroundColor Green
