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

$OPEN_WEBUI_HOST = if ($configVars["OPEN_WEBUI_HOST"]) { $configVars["OPEN_WEBUI_HOST"] } else { "localhost" }
$OPEN_WEBUI_PORT = if ($configVars["OPEN_WEBUI_PORT"]) { [int]$configVars["OPEN_WEBUI_PORT"] } else { 3000 }

Write-Host "Parsed or default OPEN_WEBUI_HOST = $OPEN_WEBUI_HOST"
Write-Host "Parsed or default OPEN_WEBUI_PORT = $OPEN_WEBUI_PORT"

Write-Host "Detecting LAN IP..."
$ipAddress = Get-IPAddress
Write-Host "IP address is: $ipAddress"
Write-Host

ShowWindowsInfo
CheckWSLFeature
CheckWindowsFirewall

Write-Host "=== Checking connectivity via config settings ($($OPEN_WEBUI_HOST):$($OPEN_WEBUI_PORT)) ==="
TestOpenWebUIPort -TargetHost $OPEN_WEBUI_HOST -TargetPort $OPEN_WEBUI_PORT
CheckOpenWebUIHTTP -Url ("http://{0}:{1}" -f $OPEN_WEBUI_HOST, $OPEN_WEBUI_PORT)
CurlOpenWebUI -Url ("http://{0}:{1}" -f $OPEN_WEBUI_HOST, $OPEN_WEBUI_PORT)

Write-Host "=== Checking connectivity via IP ($($ipAddress):$($OPEN_WEBUI_PORT)) ==="
TestOpenWebUIPort -TargetHost $ipAddress -TargetPort $OPEN_WEBUI_PORT
CheckOpenWebUIHTTP -Url ("http://{0}:{1}" -f $ipAddress, $OPEN_WEBUI_PORT)
CurlOpenWebUI -Url ("http://{0}:{1}" -f $ipAddress, $OPEN_WEBUI_PORT)

Show-PortProxyRules

Write-Host "=== END WINDOWS DIAGNOSTICS ==="
