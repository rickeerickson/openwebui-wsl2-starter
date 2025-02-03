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
$configFilePath = "$repoRoot\update_open-webui.config.sh"

Write-Host "=== Reading Bash config from: $($configFilePath) ==="
$configVars = ParseBashConfig -FilePath $configFilePath

$OPEN_WEBUI_PORT = if ($configVars["OPEN_WEBUI_PORT"]) { [int]$configVars["OPEN_WEBUI_PORT"] } else { 3000 }

Write-Log "Setting up WSL and Ubuntu..." -ForegroundColor Cyan
Request-AdminPrivileges
Install-WslIfNeeded
Set-WslVersionIfNeeded
Install-WslDistributionInteractive -DistroName "Ubuntu"

Write-Log "Starting OpenWebUI setup..." -ForegroundColor Cyan
Stop-Wsl

$wslScriptConfigPath = Convert-ToPath -WindowsPath "$PSScriptRoot\update_open-webui.config.sh"
Set-ExecutableAttribute -Path $wslScriptConfigPath

$wslScriptPath = Convert-ToPath -WindowsPath "$PSScriptRoot\update_open-webui.sh"
Set-ExecutableAttribute -Path $wslScriptPath

Start-WslScript -Path $wslScriptPath

$ipAddress = Get-IPAddress
Enable-OpenWebUIPortProxyIfNeeded -ListenAddress $ipAddress -ListenPort $OPEN_WEBUI_PORT -ConnectAddress "127.0.0.1" -ConnectPort $OPEN_WEBUI_PORT

Write-Host "Launching WSL interactively with Docker status..." -ForegroundColor Cyan
wsl -e bash -c "docker ps; exec bash"
