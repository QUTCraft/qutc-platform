[CmdletBinding()]
param(
    [string]$ComposeEnvPath = "",
    [string]$OutputRoot = "",
    [switch]$SkipMedia,
    [switch]$NoPauseWrites
)

$ErrorActionPreference = "Stop"
$repoRootPath = Split-Path -Parent $PSScriptRoot
$composePath = Join-Path $repoRootPath "deploy\compose"

if ([string]::IsNullOrWhiteSpace($ComposeEnvPath)) {
    $ComposeEnvPath = Join-Path $composePath ".env"
}
if (-not (Test-Path -LiteralPath $ComposeEnvPath)) {
    $ComposeEnvPath = Join-Path $composePath ".env.example"
}
if (-not (Test-Path -LiteralPath $ComposeEnvPath -PathType Leaf)) {
    throw "Compose environment file was not found."
}
$ComposeEnvPath = (Resolve-Path -LiteralPath $ComposeEnvPath).Path

if ([string]::IsNullOrWhiteSpace($OutputRoot)) {
    $OutputRoot = Join-Path $repoRootPath "backups"
}
if (-not (Test-Path -LiteralPath $OutputRoot)) {
    New-Item -ItemType Directory -Path $OutputRoot -Force | Out-Null
}
$OutputRoot = (Resolve-Path -LiteralPath $OutputRoot).Path

function Read-Settings {
    param([string]$Path)
    $values = @{}
    foreach ($rawLine in Get-Content -LiteralPath $Path) {
        $line = $rawLine.Trim()
        if ($line.Length -eq 0 -or $line.StartsWith("#")) {
            continue
        }
        $separator = $line.IndexOf("=")
        if ($separator -gt 0) {
            $values[$line.Substring(0, $separator)] = $line.Substring($separator + 1)
        }
    }
    return $values
}

function Invoke-Docker {
    param(
        [string[]]$Arguments,
        [switch]$Capture
    )
    $output = & docker @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "docker $($Arguments -join ' ') failed with exit code $LASTEXITCODE."
    }
    if ($Capture) {
        return @($output)
    }
}

function Invoke-Compose {
    param(
        [string[]]$Arguments,
        [switch]$Capture
    )
    $allArguments = @("compose", "--env-file", $ComposeEnvPath) + $Arguments
    return Invoke-Docker -Arguments $allArguments -Capture:$Capture
}

function Get-ContainerID {
    param([string]$Service)
    $result = Invoke-Compose -Arguments @("ps", "-q", $Service) -Capture
    $id = ($result | Out-String).Trim()
    if ([string]::IsNullOrWhiteSpace($id)) {
        throw "Compose service '$Service' is not running."
    }
    return $id
}

function Get-DatabaseTables {
    $command = 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -N -B -uroot "$MYSQL_DATABASE" -e "SHOW TABLES"'
    return @(Invoke-Compose -Arguments @("exec", "-T", "mysql", "sh", "-c", $command) -Capture | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
}

function Get-TableCount {
    param([string]$Table)
    if ($Table -notmatch "^[A-Za-z0-9_]+$") {
        throw "Unsafe table name returned by MySQL: $Table"
    }
    $command = 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -N -B -uroot "$MYSQL_DATABASE" -e "SELECT COUNT(*) FROM \`$1\`"'
    $result = Invoke-Compose -Arguments @("exec", "-T", "mysql", "sh", "-c", $command, "sh", $Table) -Capture
    $raw = ($result | Out-String).Trim()
    $count = 0L
    if (-not [long]::TryParse($raw, [ref]$count)) {
        throw "Unable to read row count for table '$Table'."
    }
    return $count
}

$settings = Read-Settings -Path $ComposeEnvPath
$databaseName = if ($settings.ContainsKey("MYSQL_DATABASE") -and $settings["MYSQL_DATABASE"]) { $settings["MYSQL_DATABASE"] } else { "qutcraft" }
$storageDriver = if ($settings.ContainsKey("STORAGE_DRIVER") -and $settings["STORAGE_DRIVER"]) { $settings["STORAGE_DRIVER"].ToLowerInvariant() } else { "local" }
if ($databaseName -notmatch "^[A-Za-z0-9_]+$") {
    throw "MYSQL_DATABASE contains unsupported characters."
}
if (-not $SkipMedia -and $storageDriver -ne "local") {
    throw "This script only archives the local media volume. For STORAGE_DRIVER=$storageDriver, use provider-level bucket versioning/backup or pass -SkipMedia for a database-only backup."
}

$stamp = [DateTime]::UtcNow.ToString("yyyyMMddTHHmmssZ")
$backupPath = Join-Path $OutputRoot "qutcraft-$stamp"
if (Test-Path -LiteralPath $backupPath) {
    throw "Backup destination already exists: $backupPath"
}
New-Item -ItemType Directory -Path $backupPath | Out-Null
$incompletePath = Join-Path $backupPath ".incomplete"
Set-Content -LiteralPath $incompletePath -Value "Backup creation has not completed." -Encoding utf8NoBOM

$containerDumpPath = "/tmp/qutcraft-$stamp.sql"
$sqlPath = Join-Path $backupPath "database.sql"
$mediaArchivePath = Join-Path $backupPath "media.tar.gz"
$mediaChecksumsPath = Join-Path $backupPath "media-files.sha256"
$apiPausedByScript = $false

Push-Location $composePath
try {
    $null = Get-ContainerID -Service "mysql"
    $apiContainerID = ""
    if (-not $SkipMedia -or -not $NoPauseWrites) {
        $apiContainerID = Get-ContainerID -Service "api"
    }
    if (-not $NoPauseWrites) {
        $apiState = (Invoke-Docker -Arguments @("inspect", $apiContainerID) -Capture | Out-String | ConvertFrom-Json)[0].State
        if (-not [bool]$apiState.Paused) {
            Invoke-Compose -Arguments @("pause", "api")
            $apiPausedByScript = $true
        }
    }

    $dumpCommand = 'umask 077; MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysqldump -uroot --single-transaction --routines --triggers --events --hex-blob --set-gtid-purged=OFF "$MYSQL_DATABASE" > "$1"'
    Invoke-Compose -Arguments @("exec", "-T", "mysql", "sh", "-c", $dumpCommand, "sh", $containerDumpPath)
    Invoke-Compose -Arguments @("cp", "mysql:$containerDumpPath", $sqlPath)

    $tableCounts = [ordered]@{}
    foreach ($table in Get-DatabaseTables) {
        $tableCounts[$table] = Get-TableCount -Table $table
    }

    $files = [ordered]@{
        "database.sql" = [ordered]@{
            sha256 = (Get-FileHash -LiteralPath $sqlPath -Algorithm SHA256).Hash.ToLowerInvariant()
            size_bytes = (Get-Item -LiteralPath $sqlPath).Length
        }
    }

    if (-not $SkipMedia) {
        $container = (Invoke-Docker -Arguments @("inspect", $apiContainerID) -Capture | Out-String | ConvertFrom-Json)[0]
        $mediaMount = @($container.Mounts | Where-Object { $_.Destination -eq "/tmp/qutcraft-uploads" }) | Select-Object -First 1
        if ($null -eq $mediaMount -or $mediaMount.Type -ne "volume" -or [string]::IsNullOrWhiteSpace($mediaMount.Name)) {
            throw "The API local media volume could not be resolved."
        }

        Invoke-Docker -Arguments @(
            "run", "--rm",
            "--mount", "type=volume,src=$($mediaMount.Name),dst=/data,readonly",
            "--mount", "type=bind,src=$backupPath,dst=/backup",
            "alpine:3.22", "tar", "-C", "/data", "-czf", "/backup/media.tar.gz", "."
        )
        $checksumCommand = 'cd /data && find . -type f -exec sha256sum "{}" \; | LC_ALL=C sort'
        $mediaChecksums = Invoke-Docker -Arguments @(
            "run", "--rm",
            "--mount", "type=volume,src=$($mediaMount.Name),dst=/data,readonly",
            "alpine:3.22", "sh", "-c", $checksumCommand
        ) -Capture
        Set-Content -LiteralPath $mediaChecksumsPath -Value @($mediaChecksums) -Encoding utf8NoBOM

        $files["media.tar.gz"] = [ordered]@{
            sha256 = (Get-FileHash -LiteralPath $mediaArchivePath -Algorithm SHA256).Hash.ToLowerInvariant()
            size_bytes = (Get-Item -LiteralPath $mediaArchivePath).Length
        }
        $files["media-files.sha256"] = [ordered]@{
            sha256 = (Get-FileHash -LiteralPath $mediaChecksumsPath -Algorithm SHA256).Hash.ToLowerInvariant()
            size_bytes = (Get-Item -LiteralPath $mediaChecksumsPath).Length
        }
    }

    $manifest = [ordered]@{
        schema = "qutc.backup/v1"
        created_at = [DateTime]::UtcNow.ToString("o")
        source_database = $databaseName
        storage_driver = $storageDriver
        includes_media = -not $SkipMedia
        writes_paused = -not $NoPauseWrites
        database_tables = $tableCounts
        files = $files
    }
    $manifest | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath (Join-Path $backupPath "manifest.json") -Encoding utf8NoBOM
    Remove-Item -LiteralPath $incompletePath

    Write-Host "BACKUP_OK: $backupPath"
    Write-Host "Database tables: $($tableCounts.Count); media included: $(-not $SkipMedia)"
}
finally {
    try {
        Invoke-Compose -Arguments @("exec", "-T", "mysql", "rm", "-f", $containerDumpPath)
    }
    catch {
        Write-Warning "Unable to remove the temporary dump inside the MySQL container: $($_.Exception.Message)"
    }
    if ($apiPausedByScript) {
        try {
            Invoke-Compose -Arguments @("unpause", "api")
        }
        catch {
            Write-Warning "Unable to unpause the API service: $($_.Exception.Message)"
        }
    }
    Pop-Location
}
