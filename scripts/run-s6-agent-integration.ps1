[CmdletBinding()]
param(
    [string]$ComposeEnvPath = "",
    [string]$ApiUrl = ""
)

$ErrorActionPreference = "Stop"
$repoRootPath = Split-Path -Parent $PSScriptRoot
if ([string]::IsNullOrWhiteSpace($ComposeEnvPath)) {
    $ComposeEnvPath = Join-Path $repoRootPath "deploy\compose\.env"
}
if (-not (Test-Path -LiteralPath $ComposeEnvPath)) {
    $ComposeEnvPath = Join-Path $repoRootPath "deploy\compose\.env.example"
}
if (-not (Test-Path -LiteralPath $ComposeEnvPath)) {
    throw "Compose environment file was not found."
}

$settings = @{}
foreach ($rawLine in Get-Content -LiteralPath $ComposeEnvPath) {
    $line = $rawLine.Trim()
    if ($line.Length -eq 0 -or $line.StartsWith("#")) {
        continue
    }
    $separator = $line.IndexOf("=")
    if ($separator -lt 1) {
        continue
    }
    $settings[$line.Substring(0, $separator)] = $line.Substring($separator + 1)
}

function Get-Setting {
    param([string]$Name, [string]$Fallback)
    if ($settings.ContainsKey($Name) -and -not [string]::IsNullOrWhiteSpace($settings[$Name])) {
        return $settings[$Name]
    }
    return $Fallback
}

$apiPort = Get-Setting "API_PORT" "8080"
if ([string]::IsNullOrWhiteSpace($ApiUrl)) {
    $ApiUrl = "http://127.0.0.1:$apiPort"
}
$mysqlUser = Get-Setting "MYSQL_USER" "qutcraft"
$mysqlPassword = Get-Setting "MYSQL_PASSWORD" "qutcraft"
$mysqlDatabase = Get-Setting "MYSQL_DATABASE" "qutcraft"
$mysqlPort = Get-Setting "MYSQL_PORT" "3306"
$redisPort = Get-Setting "REDIS_PORT" "6379"

$env:QUTC_INTEGRATION_API_URL = $ApiUrl.TrimEnd("/")
$env:QUTC_INTEGRATION_MYSQL_DSN = "${mysqlUser}:${mysqlPassword}@tcp(127.0.0.1:${mysqlPort})/${mysqlDatabase}?charset=utf8mb4&parseTime=True&loc=UTC"
$env:QUTC_INTEGRATION_REDIS_ADDR = "127.0.0.1:$redisPort"
$env:QUTC_INTEGRATION_ADMIN_EMAIL = Get-Setting "BOOTSTRAP_ADMIN_EMAIL" "admin@qutcraft.local"
$env:QUTC_INTEGRATION_ADMIN_PASSWORD = Get-Setting "BOOTSTRAP_ADMIN_PASSWORD" "change-this-development-password"
$env:QUTC_INTEGRATION_ORGANIZATION_SLUG = Get-Setting "DEFAULT_ORGANIZATION_SLUG" "qutcraft"
$env:QUTC_INTEGRATION_CACHE_NAMESPACE = Get-Setting "APP_ENV" "development"

$apiProjectPath = Join-Path $repoRootPath "apps\api"
Push-Location $apiProjectPath
try {
    & go test -tags=integration ./integration -run "^TestS6" -count=1 -v
    if ($LASTEXITCODE -ne 0) {
        throw "S6 AI agent integration tests failed with exit code $LASTEXITCODE."
    }
}
finally {
    Pop-Location
}
