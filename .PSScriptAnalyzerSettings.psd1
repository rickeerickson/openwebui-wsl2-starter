@{
    ExcludeRules = @(
        # All scripts are interactive console tools for setup
        # and diagnostics. Write-Host is intentional.
        'PSAvoidUsingWriteHost'

        # Invoke-Expression is used to run dynamically built
        # commands in the retry and config-parsing logic.
        'PSAvoidUsingInvokeExpression'

        # ShouldProcess is unnecessary for non-interactive
        # automation scripts that run unattended.
        'PSUseShouldProcessForStateChangingFunctions'

        # Function names follow established conventions
        # (e.g., Show-PortProxyRules, Remove-NetshBindings).
        'PSUseSingularNouns'

        # False positives for parameters consumed indirectly.
        'PSReviewUnusedParameter'

        # Write-Log overrides the built-in intentionally to
        # add leveled logging with file output.
        'PSAvoidOverwritingBuiltInCmdlets'
    )
}
