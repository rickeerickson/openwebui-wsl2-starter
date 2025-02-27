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

function Enable-WindowsFeatureIfNeeded {
    Enable-WindowsFeatureIfNeeded -FeatureName "Microsoft-Windows-Subsystem-Linux"
    Enable-WindowsFeatureIfNeeded -FeatureName "VirtualMachinePlatform"
}

function Install-WslIfNeeded {
    & wsl.exe --status 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "WSL is installed."
    } else {
        Write-Host "Installing WSL..."
        Start-CommandWithRetry "wsl --install -d Ubuntu --no-launch"
    }
}

function Update-Wsl {
    Write-Log "Updating WSL..." $LEVEL_INFO
    Start-CommandWithRetry "wsl --update"
}


function Set-WslVersionIfNeeded {
    Write-Log "Ensuring WSL 2 is the default version..." $LEVEL_INFO
    if (-not (wsl --list --verbose 2>&1 | Select-String "Version: 2")) {
        Start-CommandWithRetry "wsl --set-default-version 2"
    }
}

function Install-WslDistributionInteractive {
    param (
        [string]$DistroName = "Ubuntu"
    )
    
    # Check if the distro is already installed
    wsl --list | Select-String $DistroName
    if ($LASTEXITCODE -eq 0) {
        Write-Log "WSL distro '$DistroName' is already installed." $LEVEL_INFO
        Write-Log "Setting '$DistroName' as the default WSL distribution." $LEVEL_INFO
        wsl --setdefault $DistroName
        return
    }
    else {
        Write-Log "Installing WSL distro '$DistroName'..." $LEVEL_INFO
        
        $inverseBackground = $Host.UI.RawUI.ForegroundColor
        $inverseForeground = $Host.UI.RawUI.BackgroundColor

        $border = '===================================================================================================='
        $borderLength = $border.Length
        Write-Host $border -ForegroundColor $inverseForeground -BackgroundColor $inverseBackground
        Write-Host 'IMPORTANT: Ubuntu has been installed and set as the default WSL distribution.'.PadRight($borderLength) -ForegroundColor $inverseForeground -BackgroundColor $inverseBackground
        Write-Host 'Please launch Ubuntu to configure your default username and password.'.PadRight($borderLength) -ForegroundColor $inverseForeground -BackgroundColor $inverseBackground
        Write-Host 'Once you have completed the configuration, from the terminal run: `exit`'.PadRight($borderLength) -ForegroundColor $inverseForeground -BackgroundColor $inverseBackground
        Write-Host '$ exit'.PadRight($borderLength) -ForegroundColor $inverseForeground -BackgroundColor $inverseBackground
        Write-Host 'After exiting Ubuntu, please re-run the RUNME script.'.PadRight($borderLength) -ForegroundColor $inverseForeground -BackgroundColor $inverseBackground
        Write-Host $border -ForegroundColor $inverseForeground -BackgroundColor $inverseBackground
        
        wsl --install -d $DistroName

        Write-Log "Setting '$DistroName' as the default WSL distribution." $LEVEL_INFO
        wsl --setdefault $DistroName
        
        Write-Host $border -ForegroundColor $inverseForeground -BackgroundColor $inverseBackground
        Write-Host "Please re-run the RUNME script to continue.".PadRight($borderLength) -ForegroundColor $inverseForeground -BackgroundColor $inverseBackground
        Write-Host $border -ForegroundColor $inverseForeground -BackgroundColor $inverseBackground
        Exit
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
    if ($LASTEXITCODE -ne 0) {
        Write-Log "Error: The WSL script exited with code $LASTEXITCODE. Please re-run the 'RUNME' script.  If the issue continues, please report an issue in GitHub."
        exit $LASTEXITCODE
    }
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
        [string]$ListenAddress = 0.0.0.0,
        [int]$ListenPort = 3000,
        [string]$ConnectAddress = "127.0.0.1",
        [int]$ConnectPort = 3000
    )

    Write-Host "=== Checking if port proxy from $($ListenAddress):$($ListenPort) to $($ConnectAddress):$($ConnectPort) is needed ==="

    # Grab existing portproxy rules
    # Example line format: "Listen on ipv4:             Connect to ipv4:"
    # "0.0.0.0:3000        127.0.0.1:3000"
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
        # e.g. line: "0.0.0.0:3000   127.0.0.1:3000"
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

Set-StrictMode -Version Latest

function ParseBashConfig {
    param(
        [string]$FilePath
    )
    $result = @{}
    if (-not (Test-Path $FilePath)) {
        Write-Error "Config file not found: $FilePath"
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

function Show-PortProxyErrors {
    <#
    .SYNOPSIS
         Checks for conflicting port proxy rules.
    .DESCRIPTION
         Retrieves the current netsh portproxy rules and, optionally for a given Port (if provided),
         filters the rules to only that port. It then compares the listen address(es) to the current valid IP
         (via Get-IPAddress) and flags any rule whose listen address does not match the valid IP. For each invalid rule,
         it prints a remediation command.
    .EXAMPLE
         Show-PortProxyErrors -ConfigVars $configVars -Port ([int]$configVars["OLLAMA_PORT"])
    #>
    param(
         [hashtable]$ConfigVars,
         [int]$Port = $null
    )
    Write-Host "=== Checking port proxy rules for conflicts ==="

    $output = netsh interface portproxy show v4tov4 2>&1

    # Split output into lines and process only those that begin with a digit.
    $lines = $output -split "`r?`n" | ForEach-Object { $_.Trim() } | Where-Object { $_ -match '^\d' }
    $rules = @()

    $pattern = '^(?<listen>\d{1,3}(?:\.\d{1,3}){3})\s+(?<lport>\d+)\s+(?<connect>\d{1,3}(?:\.\d{1,3}){3})\s+(?<cport>\d+)$'
    foreach ($line in $lines) {
         if ($line -match $pattern) {
              $rule = [pscustomobject]@{
                   ListenAddress  = $Matches['listen']
                   ListenPort     = [int]$Matches['lport']
                   ConnectAddress = $Matches['connect']
                   ConnectPort    = [int]$Matches['cport']
                   Line           = $line
              }
              $rules += $rule
         }
    }
    if ($rules.Count -eq 0) {
         Write-Host "No port proxy rules found."
         return
    }
    
    if ($Port) {
         # Force the result into an array.
         $targetRules = @($rules | Where-Object { $_.ListenPort -eq $Port })
         if ($targetRules.Count -eq 0) {
              Write-Host "No port proxy rules found for port $Port."
              return
         }
         Test-PortGroupConflicts -Rules $targetRules -TargetPort $Port
    }
    else {
         # If no port is specified, group by ListenPort and check each.
         $groups = $rules | Group-Object -Property ListenPort
         foreach ($group in $groups) {
            Test-PortGroupConflicts -Rules $group.Group -TargetPort $group.Name
         }
    }
}

function Test-PortGroupConflicts {
    param(
        [array]$Rules,
        [Parameter(Mandatory=$true)]
        [int]$TargetPort
    )
    $validIP = Get-IPAddress
    $distinctListen = @($Rules | Select-Object -ExpandProperty ListenAddress -Unique)
    if ($distinctListen.Count -le 1) {
         Write-Host "No conflicting port proxy rules detected for port $TargetPort."
    } else {
         Write-Host "Conflict detected for port $TargetPort. Expected valid ListenAddress is $validIP." -ForegroundColor Red
         foreach ($rule in $Rules) {
              if ($rule.ListenAddress -ne $validIP) {
                   Write-Host "  Invalid rule: $($rule.Line)" -ForegroundColor Red
                   Write-Host "    To remove this rule, run:" -ForegroundColor Yellow
                   Write-Host "      netsh interface portproxy delete v4tov4 listenaddress=$($rule.ListenAddress) listenport=$TargetPort" -ForegroundColor Yellow
              } else {
                   Write-Host "  Valid rule: $($rule.Line)" -ForegroundColor Green
              }
         }
    }
}
