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