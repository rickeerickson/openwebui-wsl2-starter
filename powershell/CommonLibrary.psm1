Set-StrictMode -Version Latest

$LEVEL_ERROR = 1
$LEVEL_WARNING = 2
$LEVEL_INFO = 3
$LEVEL_DEBUG = 4
$VERBOSITY_DEFAULT = 3
$VERBOSITY = $VERBOSITY_DEFAULT

function Write-Log {
    param (
        [string]$Message,
        [int]$Level = $LEVEL_INFO
    )

    if ($Level -le $VERBOSITY) {
        $timestamp = Get-Date -Format "yyyy.MM.dd:HH:mm:ss"
        $levelPrefix = switch ($Level) {
            $LEVEL_ERROR   { "ERROR:" }
            $LEVEL_WARNING { "WARNING:" }
            $LEVEL_INFO    { "INFO:" }
            $LEVEL_DEBUG   { "DEBUG:" }
            default        { "LOG:" }
        }

        $logMessage = "$timestamp - $levelPrefix $Message"

        $callerScript = (Get-PSCallStack)[1].ScriptName
        $callerScript = if ($callerScript) { $callerScript } else { "UnknownScript.ps1" }
        $logFile = "${callerScript}.log"

        $logMessage | Tee-Object -FilePath $logFile -Append
    }
}

function Start-CommandWithRetry {
    param (
        [string]$Command,
        [bool]$ShouldFail = $false,
        [bool]$IgnoreExitStatus = $false,
        [int]$MaxRetries = 5
    )
    $retryCount = 0
    $fib1 = 10
    $fib2 = 10

    while ($true) {
        Write-Log "Running command: $Command" $LEVEL_INFO
        try {
            $output = Invoke-Expression -Command $Command 2>&1
            $exitCode = $LASTEXITCODE

            Write-Log "Command Output: $output" $LEVEL_DEBUG

            if ($ShouldFail -and $exitCode -eq 0) {
                throw "Command succeeded unexpectedly: $Command"
            }

            if (-not $ShouldFail -and $exitCode -ne 0) {
                throw "Command failed with exit code ${exitCode}: ${Command}"
            }

            if ($IgnoreExitStatus) {
                Write-Log "Exit code ignored for command: $Command" $LEVEL_WARNING
                return
            }

            Write-Log "Command executed successfully: $Command" $LEVEL_INFO
            break
        }
        catch {
            Write-Log "Error: $_" $LEVEL_WARNING

            if ($retryCount -ge $MaxRetries) {
                Write-Log "Maximum retries reached. Aborting command: $Command" $LEVEL_ERROR
                if (-not $IgnoreExitStatus) {
                    exit 1
                }
            }

            $retryCount++
            Write-Log "Retrying in $fib1 seconds... (Retry $retryCount/$MaxRetries)" $LEVEL_WARNING
            Start-Sleep -Seconds $fib1
            $newDelay = $fib1 + $fib2
            $fib1 = $fib2
            $fib2 = $newDelay
        }
    }
}

function Request-AdminPrivileges {
    if (-not ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
        Write-Log "Requesting elevated privileges..." $LEVEL_WARNING
        Start-Process powershell.exe "-ExecutionPolicy Bypass -File `"$MyInvocation.MyCommand.Definition`"" -Verb RunAs
        exit
    }
}

function Enable-WindowsFeatureIfNeeded {
    param ([string]$FeatureName)

    $feature = Get-WindowsOptionalFeature -FeatureName $FeatureName -Online
    if ($feature.State -ne "Enabled") {
        Write-Log "Enabling feature: $FeatureName..." $LEVEL_INFO
        Start-CommandWithRetry "Enable-WindowsOptionalFeature -Online -FeatureName $FeatureName -NoRestart"
        Write-Log "Feature '$FeatureName' has been enabled. Please reboot the system." $LEVEL_WARNING
        exit
    }
    Write-Log "Feature '$FeatureName' is already enabled." $LEVEL_INFO
}

function Install-WslIfNeeded {
    Enable-WindowsFeatureIfNeeded -FeatureName "Microsoft-Windows-Subsystem-Linux"
    Enable-WindowsFeatureIfNeeded -FeatureName "VirtualMachinePlatform"
}

function Set-WslVersionIfNeeded {
    Write-Log "Ensuring WSL 2 is the default version..." $LEVEL_INFO
    if (-not (wsl --list --verbose 2>&1 | Select-String "WSL 2")) {
        Start-CommandWithRetry "wsl --set-default-version 2"
    }
}

function Install-WslDistroIfNeeded {
    param ([string]$DistroName = "Ubuntu")

    $wslDistros = wsl --list --quiet

    if (-not ($wslDistros -contains $DistroName)) {
        Write-Log "Installing WSL distro: $DistroName..." $LEVEL_INFO
        Start-CommandWithRetry { wsl --install -d $DistroName }
    }
    else {
        Write-Log "WSL distro '$DistroName' is already installed." $LEVEL_INFO
    }
}

function Stop-Wsl {
    Write-Log "Stopping WSL..." $LEVEL_INFO
    Start-CommandWithRetry "wsl --shutdown"
}

function Update-Wsl {
    Stop-Wsl
    Write-Log "Updating WSL..." $LEVEL_INFO
    Start-CommandWithRetry "wsl --update"
}

function Remove-WslDistro {
    param ([string]$DistroName)
    Stop-Wsl
    Write-Log "Unregistering WSL distro '$DistroName'..." $LEVEL_INFO
    Start-CommandWithRetry "wsl --unregister $DistroName"
}

function Convert-ToPath {
    param (
        [string]$WindowsPath
    )
    $resolvedPath = Resolve-Path -Path $WindowsPath
    return "/mnt/" + ($resolvedPath.Path -replace '\\', '/' -replace ':', '').ToLower()
}

function Set-ExecutableAttribute {
    param (
        [string]$Path
    )
    Start-CommandWithRetry "wsl chmod +x $Path"
}

function Start-WslScript {
    param (
        [string]$Path
    )
    Write-Log "Running script: ${Path}..." -ForegroundColor Cyan
    wsl bash -c "$Path"
}

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
        if ($line -match '^\s*#' -or $line -match '^\s*$') {
            continue
        }
        if ($line -match '^\s*([A-Z0-9_]+)=(.*)$') {
            $varName = $Matches[1]
            $varValue = $Matches[2].Trim()
            $varValue = $varValue -replace '^["''](.*)["'']$', '$1'
            $result[$varName] = $varValue
        }
    }
    return $result
}

function Get-IPAddress {
    $defaultRoute = Get-NetRoute -DestinationPrefix "0.0.0.0/0" -ErrorAction SilentlyContinue |
        Sort-Object -Property RouteMetric,InterfaceMetric |
        Select-Object -First 1

    if (-not $defaultRoute) {
        Write-Host "Could not find a default IPv4 route. Using 'localhost' as fallback."
        return "localhost"
    }

    $interfaceIndex = $defaultRoute.InterfaceIndex
    $addresses = Get-NetIPAddress -AddressFamily IPv4 -InterfaceIndex $interfaceIndex -ErrorAction SilentlyContinue |
        Where-Object {
            $_.IPAddress -ne "127.0.0.1" -and -not $_.IPAddress.StartsWith("169.254")
        }

    if ($addresses) {
        $first = $addresses | Select-Object -First 1
        return $first.IPAddress
    } else {
        Write-Host "No valid IPv4 address found on the default route interface. Using 'localhost' as fallback."
        return "localhost"
    }
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
  Write-Host "=== Test TCP on $($TargetHost):$($TargetPort) ==="
  $connection = Test-NetConnection -ComputerName $TargetHost -Port $TargetPort -WarningAction SilentlyContinue
  if ($connection.TcpTestSucceeded) {
    Write-Host "TCP connection to port $($TargetPort) on $($TargetHost) succeeded!"
    Write-Host "Remote Address:" $connection.RemoteAddress
  } else {
    Write-Host "TCP connection to port $($TargetPort) on $($TargetHost) failed."
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

function Enable-OpenWebUIPortProxy {
    param(
        [string]$ListenAddress,
        [int]$ListenPort = 3000,
        [string]$ConnectAddress = "127.0.0.1",
        [int]$ConnectPort = 3000
    )

    Write-Host "=== Enabling port proxy from $($ListenAddress):$($ListenPort) to $($ConnectAddress):$($ConnectPort) ==="
    # Remove any existing proxy rule on that address:port
    netsh interface portproxy delete v4tov4 `
        listenaddress=$ListenAddress `
        listenport=$ListenPort `
        2>$null | Out-Null

    # Create the new rule
    netsh interface portproxy add v4tov4 `
        listenaddress=$ListenAddress `
        listenport=$ListenPort `
        connectaddress=$ConnectAddress `
        connectport=$ConnectPort `
        2>$null

    $ruleName = "OpenWebUI-$($ListenAddress)-$($ListenPort)"
    if (-not (Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue)) {
        Write-Host "Creating firewall rule for $($ListenAddress):$($ListenPort)"
        New-NetFirewallRule `
            -DisplayName $ruleName `
            -Direction Inbound `
            -Protocol TCP `
            -LocalPort $ListenPort `
            -Action Allow `
            | Out-Null
    } else {
        Write-Host "Firewall rule already exists: $ruleName"
    }

    Write-Host "Port proxy setup complete."
    Write-Host
}

function Show-PortProxyRules {
    Write-Host "=== Existing netsh portproxy rules ==="
    # This is read-only: it just lists them
    $proxyRules = netsh interface portproxy show v4tov4 2>$null
    if ($proxyRules) {
        $proxyRules
    } else {
        Write-Host "No v4tov4 portproxy rules found."
    }
    Write-Host
}

function Enable-OpenWebUIPortProxyIfNeeded {
    param(
        [string]$ListenAddress,
        [int]$ListenPort = 3000,
        [string]$ConnectAddress = "127.0.0.1",
        [int]$ConnectPort = 3000
    )

    Write-Host "=== Checking if port proxy from $($ListenAddress):$($ListenPort) to $($ConnectAddress):$($ConnectPort) is needed ==="

    # Grab existing portproxy rules
    # Example line format: "Listen on ipv4:             Connect to ipv4:"
    # "192.168.0.10:3000        127.0.0.1:3000"
    $rules = netsh interface portproxy show v4tov4 2>$null

    if (-not $rules) {
        Write-Host "No existing port proxy rules found or command failed."
        $rules = @() # empty array to handle below logic
    } else {
        $rules = $rules -split "`r?`n" | ForEach-Object { $_.Trim() } | Where-Object { $_ -match '^\S+:\d+\s+\S+:\d+' }
    }

    # This pattern looks for lines with "ListenAddress:ListenPort  ConnectAddress:ConnectPort"
    # We'll see if there's a rule that EXACTLY matches our intended connection
    $ruleExists = $false
    $foundDifferent = $false

    foreach ($line in $rules) {
        # e.g. line: "192.168.0.10:3000   127.0.0.1:3000"
        if ($line -match '^(?<listen>[^:]+):(?<lport>\d+)\s+(?<connect>[^:]+):(?<cport>\d+)$') {
            $existingListenAddr = $Matches['listen']
            $existingListenPort = $Matches['lport']
            $existingConnectAddr = $Matches['connect']
            $existingConnectPort = $Matches['cport']

            if ($existingListenAddr -eq $ListenAddress -and [int]$existingListenPort -eq $ListenPort) {
                # We found a rule that listens on the same address/port
                if ($existingConnectAddr -eq $ConnectAddress -and [int]$existingConnectPort -eq $ConnectPort) {
                    $ruleExists = $true
                } else {
                    # Same listen but different connect
                    $foundDifferent = $true
                }
            }
        }
    }

    if ($ruleExists -and -not $foundDifferent) {
        Write-Host "Port proxy rule already exists. No changes needed."
    } else {
        if ($foundDifferent) {
            Write-Host "Different rule found for $($ListenAddress):$($ListenPort). Removing old rule..."
            netsh interface portproxy delete v4tov4 `
                listenaddress=$ListenAddress `
                listenport=$ListenPort `
                2>$null | Out-Null
        }

        Write-Host "Creating port proxy from $($ListenAddress):$($ListenPort) to $($ConnectAddress):$($ConnectPort)..."
        netsh interface portproxy add v4tov4 `
            listenaddress=$ListenAddress `
            listenport=$ListenPort `
            connectaddress=$ConnectAddress `
            connectport=$ConnectPort `
            2>$null

        Write-Host "Port proxy setup complete."
    }

    $ruleName = "OpenWebUI-$($ListenAddress)-$($ListenPort)"
    $firewallRule = Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue

    if (-not $firewallRule) {
        Write-Host "Creating firewall rule for $($ListenAddress):$($ListenPort)"
        New-NetFirewallRule `
            -DisplayName $ruleName `
            -Direction Inbound `
            -Protocol TCP `
            -LocalPort $ListenPort `
            -Action Allow `
            | Out-Null
    } else {
        Write-Host "Firewall rule already exists: $ruleName"
    }

    Write-Host "=== Done. ==="
}
