@echo off
setlocal EnableDelayedExpansion

set PASS_COUNT=0
set FAIL_COUNT=0
set SKIP_COUNT=0

echo.
echo =======================================
echo  build_and_test.cmd
echo =======================================

REM --------------------------------------------------------------------------
REM Check for PowerShell
REM --------------------------------------------------------------------------
where pwsh >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo.
    echo  SKIP: pwsh not found, install PowerShell 7+
    echo         https://github.com/PowerShell/PowerShell
    set /a SKIP_COUNT+=1
    goto :Summary
)

REM --------------------------------------------------------------------------
REM Syntax: PowerShell parse
REM --------------------------------------------------------------------------
echo.
echo === Syntax: PowerShell parse ===

for %%F in (
    "RUNME.ps1"
    "powershell\CommonLibrary.psm1"
    "powershell\Install-Ubuntu.ps1"
    "powershell\Remove-NetShBindings.ps1"
    "powershell\Remove-Ubuntu.ps1"
    "powershell\Update-Wsl2.ps1"
    "ollama\scripts\Test-OllamaHealth.ps1"
    "open-webui\scripts\Test-OpenWebUIHealth.ps1"
) do (
    pwsh -NoProfile -Command ^
        "$errors = $null; [System.Management.Automation.Language.Parser]::ParseFile('%~dp0%%~F', [ref]$null, [ref]$errors) | Out-Null; if ($errors.Count -gt 0) { $errors | ForEach-Object { Write-Output $_.ToString() }; exit 1 }" >nul 2>&1
    if !ERRORLEVEL! equ 0 (
        echo   PASS: pwsh parse %%~F
        set /a PASS_COUNT+=1
    ) else (
        echo   FAIL: pwsh parse %%~F
        pwsh -NoProfile -Command ^
            "$errors = $null; [System.Management.Automation.Language.Parser]::ParseFile('%~dp0%%~F', [ref]$null, [ref]$errors) | Out-Null; $errors | ForEach-Object { Write-Output $_.ToString() }"
        set /a FAIL_COUNT+=1
    )
)

REM --------------------------------------------------------------------------
REM Lint: PSScriptAnalyzer
REM --------------------------------------------------------------------------
echo.
echo === Lint: PSScriptAnalyzer ===

pwsh -NoProfile -Command "if (Get-Module -ListAvailable PSScriptAnalyzer) { exit 0 } else { exit 1 }" >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo   SKIP: PSScriptAnalyzer module not installed
    echo         Install: pwsh -Command "Install-Module PSScriptAnalyzer -Force -Scope CurrentUser"
    set /a SKIP_COUNT+=1
    goto :AfterPSScriptAnalyzer
)

for %%F in (
    "RUNME.ps1"
    "powershell\CommonLibrary.psm1"
    "powershell\Install-Ubuntu.ps1"
    "powershell\Remove-NetShBindings.ps1"
    "powershell\Remove-Ubuntu.ps1"
    "powershell\Update-Wsl2.ps1"
    "ollama\scripts\Test-OllamaHealth.ps1"
    "open-webui\scripts\Test-OpenWebUIHealth.ps1"
) do (
    for /f %%R in ('pwsh -NoProfile -Command "$r = Invoke-ScriptAnalyzer -Path '%~dp0%%~F' -Settings '%~dp0.PSScriptAnalyzerSettings.psd1' -Severity Warning,Error; if ($r) { exit 1 } else { exit 0 }"') do (rem)
    pwsh -NoProfile -Command "$r = Invoke-ScriptAnalyzer -Path '%~dp0%%~F' -Settings '%~dp0.PSScriptAnalyzerSettings.psd1' -Severity Warning,Error; if ($r) { $r | Format-Table -AutoSize | Out-String | Write-Host; exit 1 }" >nul 2>&1
    if !ERRORLEVEL! equ 0 (
        echo   PASS: PSScriptAnalyzer %%~F
        set /a PASS_COUNT+=1
    ) else (
        echo   FAIL: PSScriptAnalyzer %%~F
        pwsh -NoProfile -Command "Invoke-ScriptAnalyzer -Path '%~dp0%%~F' -Settings '%~dp0.PSScriptAnalyzerSettings.psd1' -Severity Warning,Error | Format-Table -AutoSize"
        set /a FAIL_COUNT+=1
    )
)

:AfterPSScriptAnalyzer

REM --------------------------------------------------------------------------
REM Summary
REM --------------------------------------------------------------------------
:Summary
echo.
echo =======================================
echo   Pass: %PASS_COUNT%  Fail: %FAIL_COUNT%  Skip: %SKIP_COUNT%
echo =======================================
echo.

if %FAIL_COUNT% gtr 0 exit /b 1
exit /b 0
