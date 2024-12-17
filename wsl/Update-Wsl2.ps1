Set-StrictMode -Version Latest

if (Get-Module -Name WslCommon) {
    Remove-Module -Name WslCommon -Force
}
Import-Module "$PSScriptRoot\wsl\WslCommon.psm1" -Force

Write-Log "Updating WSL..." -ForegroundColor Cyan
Request-AdminPrivileges
Update-Wsl

Write-Log "WSL update completed successfully." -ForegroundColor Green
