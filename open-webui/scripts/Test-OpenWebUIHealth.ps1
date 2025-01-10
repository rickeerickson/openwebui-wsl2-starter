# Path to the config file (adjust as needed relative to this script's location)
$ConfigFilePath = "..\..\update_open-webui.config.sh"

function ParseBashConfig {
    param(
        [string]$FilePath
    )
    $result = @{}

    if (-not (Test-Path $FilePath)) {
        Write-Host "Config file not found: $FilePath"
        return $result
    }

    $lines = Get-Content $FilePath

    foreach ($line in $lines) {
        # Ignore comments or empty lines
        if ($line -match '^\s*#' -or $line -match '^\s*$') {
            continue
        }
        # Match lines like: VARIABLE="value" or VARIABLE=value
        if ($line -match '^\s*([A-Z0-9_]+)=(.*)$') {
            $varName = $Matches[1]
            $varValue = $Matches[2].Trim()

            # Remove surrounding quotes if present
            $varValue = $varValue -replace '^["''](.*)["'']$', '$1'
            $result[$varName] = $varValue
        }
    }
    return $result
}

function ShowWindowsInfo {
  Write-Host "=== Windows Info ==="
  Write-Host "User:" $env:UserName
  Write-Host "Computer:" $env:ComputerName
  Write-Host "OS Version:" (Get-CimInstance Win32_OperatingSystem).Caption
  Write-Host "Build:" (Get-CimInstance Win32_OperatingSystem).BuildNumber
  Write-Host
}

function CheckWSLFeature {
  Write-Host "=== WSL Feature Status ==="
  $feature = Get-WindowsOptionalFeature -Online -FeatureName Microsoft-Windows-Subsystem-Linux
  Write-Host "WSL Enabled:" $feature.State
  Write-Host
}

function CheckWindowsFirewall {
  Write-Host "=== Windows Firewall ==="
  $firewallProfiles = Get-NetFirewallProfile | Select-Object Name, Enabled
  $firewallProfiles | ForEach-Object {
    Write-Host ($_.Name + " Firewall Enabled: " + $_.Enabled)
  }
  Write-Host
}

function TestOpenWebUIPort {
  param(
    [string]$TargetHost = "localhost",
    [int]$TargetPort = 3000
  )
  Write-Host "=== Test Open-WebUI Port ($TargetPort) on $TargetHost ==="
  $connection = Test-NetConnection -ComputerName $TargetHost -Port $TargetPort -WarningAction SilentlyContinue
  if ($connection.TcpTestSucceeded) {
    Write-Host "TCP connection to port $TargetPort succeeded!"
    Write-Host "Remote Address:" $connection.RemoteAddress
  } else {
    Write-Host "TCP connection to port $TargetPort failed."
  }
  Write-Host
}

function CheckOpenWebUIHTTP {
  param(
    [string]$Url = "http://localhost:3000"
  )
  Write-Host "=== HTTP Check (Invoke-WebRequest) on $Url ==="
  try {
    $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 5
    Write-Host "HTTP Status Code:" $response.StatusCode
  } catch {
    Write-Host "Request failed or no valid HTTP response."
  }
  Write-Host
}

function CurlOpenWebUI {
  param(
    [string]$Url = "http://localhost:3000"
  )
  Write-Host "=== Raw cURL call to $Url (HTTP code only) ==="
  try {
    $rawOutput = & curl.exe -s -o NUL -w "%{http_code}" $Url 2>&1
    Write-Host "HTTP Code:" $rawOutput
  } catch {
    Write-Host "cURL call failed with error:" $_.Exception.Message
  }
  Write-Host
}

Write-Host "=== Reading Bash config from: $ConfigFilePath ==="
$configVars = ParseBashConfig -FilePath $ConfigFilePath

# Default to 'localhost' and '3000' if not found
$OPEN_WEBUI_HOST = if ($configVars["OPEN_WEBUI_HOST"]) { $configVars["OPEN_WEBUI_HOST"] } else { "localhost" }
$OPEN_WEBUI_PORT = if ($configVars["OPEN_WEBUI_PORT"]) { [int]$configVars["OPEN_WEBUI_PORT"] } else { 3000 }

Write-Host "Parsed or default OPEN_WEBUI_HOST =" $OPEN_WEBUI_HOST
Write-Host "Parsed or default OPEN_WEBUI_PORT =" $OPEN_WEBUI_PORT
Write-Host

ShowWindowsInfo
CheckWSLFeature
CheckWindowsFirewall
TestOpenWebUIPort -TargetHost $OPEN_WEBUI_HOST -TargetPort $OPEN_WEBUI_PORT
CheckOpenWebUIHTTP -Url ("http://{0}:{1}" -f $OPEN_WEBUI_HOST, $OPEN_WEBUI_PORT)
CurlOpenWebUI -Url ("http://{0}:{1}" -f $OPEN_WEBUI_HOST, $OPEN_WEBUI_PORT)
Write-Host "=== END WINDOWS DIAGNOSTICS ==="
