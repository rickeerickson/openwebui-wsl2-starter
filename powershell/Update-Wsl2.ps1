Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Get-RepoRoot {
    param(
        [string]$StartDir = $PSScriptRoot
    )

    # Ensure we have an absolute path
    $resolvedStartDir = (Resolve-Path $StartDir).Path

    # Pass the directory explicitly to Git
    $repoRoot = & git -C $resolvedStartDir rev-parse --show-toplevel 2>$null
    if (-not $repoRoot) {
        Write-Error "Not a Git repository or Git not installed."
        return $null
    }
    return $repoRoot
}

$repoRoot = Get-RepoRoot

if (Get-Module -Name CommonLibrary) {
    Remove-Module -Name CommonLibrary -Force
}

Import-Module "$repoRoot\powershell\CommonLibrary.psm1" -Force

Write-Log "Updating WSL..." -ForegroundColor Cyan
Request-AdminPrivileges
Update-Wsl

Write-Log "WSL update completed successfully." -ForegroundColor Green
