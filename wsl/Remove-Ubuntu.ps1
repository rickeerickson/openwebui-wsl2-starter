Set-StrictMode -Version Latest

if (Get-Module -Name WslCommon) {
    Remove-Module -Name WslCommon -Force
}
Import-Module "$PSScriptRoot\wsl\WslCommon.psm1" -Force

Write-Log "Removing Ubuntu WSL distribution..." -ForegroundColor Cyan
Request-AdminPrivileges
Remove-WslDistro -DistroName "Ubuntu"

Write-Log "Ubuntu WSL distribution removed successfully." -ForegroundColor Green
