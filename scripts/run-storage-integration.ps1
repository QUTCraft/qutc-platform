param(
    [string]$ComposeEnvPath = "",
    [string]$ApiUrl = "",
    [string]$Endpoint = "localhost:9000",
    [string]$AccessKey = "minioadmin",
    [string]$SecretKey = "minioadmin-change-me",
    [string]$Bucket = "qutcraft-media-test"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

$settings = @{}
if ([string]::IsNullOrWhiteSpace($ComposeEnvPath)) {
    $ComposeEnvPath = Join-Path $root "deploy\compose\.env"
}
if (-not (Test-Path -LiteralPath $ComposeEnvPath)) {
    $ComposeEnvPath = Join-Path $root "deploy\compose\.env.example"
}
foreach ($rawLine in Get-Content -LiteralPath $ComposeEnvPath) {
    $line = $rawLine.Trim()
    if ($line.Length -eq 0 -or $line.StartsWith("#")) { continue }
    $separator = $line.IndexOf("=")
    if ($separator -gt 0) {
        $settings[$line.Substring(0, $separator)] = $line.Substring($separator + 1)
    }
}
function Get-Setting([string]$Name, [string]$Fallback) {
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

$env:S3_TEST_ENDPOINT = $Endpoint
$env:S3_TEST_ACCESS_KEY = $AccessKey
$env:S3_TEST_SECRET_KEY = $SecretKey
$env:S3_TEST_BUCKET = $Bucket
$env:QUTC_INTEGRATION_API_URL = $ApiUrl.TrimEnd("/")
$env:QUTC_INTEGRATION_MYSQL_DSN = "${mysqlUser}:${mysqlPassword}@tcp(127.0.0.1:${mysqlPort})/${mysqlDatabase}?charset=utf8mb4&parseTime=True&loc=UTC"
$env:QUTC_INTEGRATION_REDIS_ADDR = "127.0.0.1:$redisPort"
$env:QUTC_INTEGRATION_ADMIN_EMAIL = Get-Setting "BOOTSTRAP_ADMIN_EMAIL" "admin@qutcraft.local"
$env:QUTC_INTEGRATION_ADMIN_PASSWORD = Get-Setting "BOOTSTRAP_ADMIN_PASSWORD" "change-this-development-password"
$env:QUTC_INTEGRATION_ORGANIZATION_SLUG = Get-Setting "DEFAULT_ORGANIZATION_SLUG" "qutcraft"
$env:QUTC_INTEGRATION_CACHE_NAMESPACE = Get-Setting "APP_ENV" "development"

Push-Location (Join-Path $root "apps/api")
try {
    go test -tags=integration ./internal/platform/storage ./integration -run "^(TestS3RoundTripAgainstMinIO|TestS5S3MediaUploadAndPublicDownload)$" -count=1 -v
    if ($LASTEXITCODE -ne 0) {
        throw "S3/MinIO integration tests failed with exit code $LASTEXITCODE."
    }
} finally {
    Pop-Location
}
