[CmdletBinding()]
param(
    [ValidateRange(1, 3)]
    [int]$RoundsPerOrganization = 1,
    [ValidateRange(10, 300)]
    [int]$PollTimeoutSeconds = 120,
    [string]$ApiUrl = "",
    [string]$EnvFile = "",
    [string]$OutputDirectory = "",
    [switch]$AllowMock
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$runner = Join-Path $PSScriptRoot "run-ai-activity-demo-rehearsal.ps1"
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $stamp = (Get-Date).ToUniversalTime().ToString("yyyyMMdd-HHmmss")
    $OutputDirectory = Join-Path $root "tmp/agent-rehearsal/multi-organization-$stamp"
}

$organizations = @(
    @{ slug = "qutcraft"; query = "服务器开放日" },
    @{ slug = "campus-commons"; query = "活动安全" }
)

foreach ($organization in $organizations) {
    $arguments = @{
        Rounds = $RoundsPerOrganization
        PollTimeoutSeconds = $PollTimeoutSeconds
        OrganizationSlug = $organization.slug
        SourceQuery = $organization.query
        OutputPath = Join-Path $OutputDirectory "$($organization.slug).json"
    }
    if (-not [string]::IsNullOrWhiteSpace($ApiUrl)) {
        $arguments.ApiUrl = $ApiUrl
    }
    if (-not [string]::IsNullOrWhiteSpace($EnvFile)) {
        $arguments.EnvFile = $EnvFile
    }
    if ($AllowMock) {
        $arguments.AllowMock = $true
    }
    & $runner @arguments
}

Write-Host "MULTI_ORGANIZATION_DEMO_REHEARSAL_OK: organizations=2 rounds_per_organization=$RoundsPerOrganization output=$OutputDirectory"
