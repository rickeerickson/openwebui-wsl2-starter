Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Get-RepoRoot {
    param(
        [string]$StartDir = $PSScriptRoot
    )

    # Ensure we have an absolute path
    $resolvedStartDir = (Resolve-Path $StartDir).Path

    # Use Git's -C option to get the repository root without changing location
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
$configVars = ParseBashConfig -FilePath $configFilePath

$wslScriptPath = Convert-ToPath -WindowsPath "$PSScriptRoot\diagnose_ollama.sh"
Set-ExecutableAttribute -Path $wslScriptPath
Start-WslScript -Path $wslScriptPath

Show-PortProxyRules

$ollamaPort = [int]$configVars["OLLAMA_PORT"]
Write-Host "=== Highlighting port proxy conflicts for OLLAMA_PORT=${ollamaPort} ==="
Show-PortProxyErrors -ConfigVars $configVars -Port $ollamaPort
