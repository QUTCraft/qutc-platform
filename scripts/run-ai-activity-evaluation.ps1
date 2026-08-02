[CmdletBinding()]
param(
    [ValidateSet("mock", "openai_compatible")]
    [string]$Provider = "mock",
    [string]$OutputPath = "",
    [string]$EnvFile = "",
    [switch]$IncludeOutput
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$dataset = Join-Path $root "docs/product/ai-activity-evaluation-cases.json"
if ([string]::IsNullOrWhiteSpace($EnvFile)) {
    $EnvFile = Join-Path $root "deploy/compose/.env"
}
if ([string]::IsNullOrWhiteSpace($OutputPath)) {
    $OutputPath = Join-Path $root "tmp/agent-evaluation/activity-planner-$Provider.json"
}

$aiVariableNames = @("AI_BASE_URL", "AI_API_KEY", "AI_MODEL", "AI_REQUEST_TIMEOUT")
$previousValues = @{}
try {
    foreach ($name in $aiVariableNames) {
        $previousValues[$name] = [Environment]::GetEnvironmentVariable($name, "Process")
    }
    if ($Provider -eq "openai_compatible" -and (Test-Path -LiteralPath $EnvFile)) {
        foreach ($line in Get-Content -LiteralPath $EnvFile) {
            if ($line -notmatch '^([A-Z0-9_]+)=(.*)$') {
                continue
            }
            $name, $value = $matches[1], $matches[2]
            if ($aiVariableNames -contains $name -and [string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($name, "Process"))) {
                [Environment]::SetEnvironmentVariable($name, $value, "Process")
            }
        }
    }
    if ($Provider -eq "openai_compatible") {
        foreach ($name in @("AI_BASE_URL", "AI_API_KEY", "AI_MODEL")) {
            if ([string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($name, "Process"))) {
                throw "$name is required for a real-model evaluation. Configure it in the current shell or ignored Compose .env; do not commit credentials."
            }
        }
    }

$arguments = @(
    "run", "./cmd/agent-eval",
    "-dataset", $dataset,
    "-output", $OutputPath,
    "-provider", $Provider,
    "-fail-under", "10"
)
if ($IncludeOutput) {
    $arguments += "-include-output"
}

    Push-Location (Join-Path $root "apps/api")
    try {
        & go @arguments
        if ($LASTEXITCODE -ne 0) {
            throw "Activity planner evaluation failed with exit code $LASTEXITCODE."
        }
    } finally {
        Pop-Location
    }
} finally {
    foreach ($name in $aiVariableNames) {
        [Environment]::SetEnvironmentVariable($name, $previousValues[$name], "Process")
    }
}

Write-Host "AI_ACTIVITY_EVALUATION_OK: report=$OutputPath"
