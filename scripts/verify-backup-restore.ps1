[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$BackupPath,
    [string]$ComposeEnvPath = "",
    [switch]$DeleteBackupAfterVerification
)

$ErrorActionPreference = "Stop"
$repoRootPath = Split-Path -Parent $PSScriptRoot
$composePath = Join-Path $repoRootPath "deploy\compose"

if (-not (Test-Path -LiteralPath $BackupPath -PathType Container)) {
    throw "Backup directory does not exist: $BackupPath"
}
$BackupPath = (Resolve-Path -LiteralPath $BackupPath).Path
if (Test-Path -LiteralPath (Join-Path $BackupPath ".incomplete")) {
    throw "Backup is marked incomplete and cannot be restored."
}

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
    return Invoke-Docker -Arguments (@("compose", "--env-file", $ComposeEnvPath) + $Arguments) -Capture:$Capture
}

function Invoke-RootSQL {
    param([string]$Statement)
    $command = 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -N -B -uroot -e "$1"'
    return Invoke-Compose -Arguments @("exec", "-T", "mysql", "sh", "-c", $command, "sh", $Statement) -Capture
}

function Get-RestoredTableCount {
    param(
        [string]$Database,
        [string]$Table
    )
    if ($Database -notmatch "^[A-Za-z0-9_]+$" -or $Table -notmatch "^[A-Za-z0-9_]+$") {
        throw "Unsafe database or table name in restore verification."
    }
    $command = 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -N -B -uroot "$1" -e "SELECT COUNT(*) FROM \`$2\`"'
    $output = Invoke-Compose -Arguments @("exec", "-T", "mysql", "sh", "-c", $command, "sh", $Database, $Table) -Capture
    $raw = ($output | Out-String).Trim()
    $count = 0L
    if (-not [long]::TryParse($raw, [ref]$count)) {
        throw "Unable to read restored row count for table '$Table'."
    }
    return $count
}

$manifestPath = Join-Path $BackupPath "manifest.json"
if (-not (Test-Path -LiteralPath $manifestPath -PathType Leaf)) {
    throw "manifest.json is missing from the backup."
}
$manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
if ($manifest.schema -ne "qutc.backup/v1") {
    throw "Unsupported backup schema: $($manifest.schema)"
}

foreach ($fileProperty in $manifest.files.PSObject.Properties) {
    $fileName = $fileProperty.Name
    if ([IO.Path]::GetFileName($fileName) -ne $fileName) {
        throw "Unsafe file name in backup manifest: $fileName"
    }
    $filePath = Join-Path $BackupPath $fileName
    if (-not (Test-Path -LiteralPath $filePath -PathType Leaf)) {
        throw "Backup file is missing: $fileName"
    }
    $actualHash = (Get-FileHash -LiteralPath $filePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualHash -ne [string]$fileProperty.Value.sha256) {
        throw "Checksum mismatch for backup file: $fileName"
    }
    if ((Get-Item -LiteralPath $filePath).Length -ne [long]$fileProperty.Value.size_bytes) {
        throw "Size mismatch for backup file: $fileName"
    }
}

$suffix = ([Guid]::NewGuid().ToString("N")).Substring(0, 12)
$restoreDatabase = "qutc_restore_verify_$suffix"
$restoreVolume = "qutcraft_restore_verify_$suffix"
$containerDumpPath = "/tmp/$restoreDatabase.sql"
$databaseCreated = $false
$volumeCreated = $false
$verificationSucceeded = $false

Push-Location $composePath
try {
    $mysqlContainerID = (Invoke-Compose -Arguments @("ps", "-q", "mysql") -Capture | Out-String).Trim()
    if ([string]::IsNullOrWhiteSpace($mysqlContainerID)) {
        throw "Compose service 'mysql' is not running."
    }

    $null = Invoke-RootSQL -Statement "CREATE DATABASE ``$restoreDatabase`` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
    $databaseCreated = $true
    Invoke-Compose -Arguments @("cp", (Join-Path $BackupPath "database.sql"), "mysql:$containerDumpPath")
    $restoreCommand = 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot "$1" < "$2"'
    Invoke-Compose -Arguments @("exec", "-T", "mysql", "sh", "-c", $restoreCommand, "sh", $restoreDatabase, $containerDumpPath)

    $expectedTables = @($manifest.database_tables.PSObject.Properties.Name | Sort-Object)
    $showCommand = 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -N -B -uroot "$1" -e "SHOW TABLES"'
    $actualTables = @(Invoke-Compose -Arguments @("exec", "-T", "mysql", "sh", "-c", $showCommand, "sh", $restoreDatabase) -Capture | Where-Object { $_ } | Sort-Object)
    if (($expectedTables -join "`n") -ne ($actualTables -join "`n")) {
        throw "Restored database table set does not match the backup manifest."
    }
    foreach ($tableProperty in $manifest.database_tables.PSObject.Properties) {
        $actualCount = Get-RestoredTableCount -Database $restoreDatabase -Table $tableProperty.Name
        if ($actualCount -ne [long]$tableProperty.Value) {
            throw "Restored row count mismatch for '$($tableProperty.Name)': got $actualCount, want $($tableProperty.Value)."
        }
    }

    if ([bool]$manifest.includes_media) {
        if ($manifest.storage_driver -ne "local") {
            throw "Media restore verification only supports local-volume backups."
        }
        $null = Invoke-Docker -Arguments @("volume", "create", $restoreVolume) -Capture
        $volumeCreated = $true
        Invoke-Docker -Arguments @(
            "run", "--rm",
            "--mount", "type=volume,src=$restoreVolume,dst=/data",
            "--mount", "type=bind,src=$BackupPath,dst=/backup,readonly",
            "alpine:3.22", "tar", "-C", "/data", "-xzf", "/backup/media.tar.gz"
        )
        $checksumCommand = 'cd /data && find . -type f -exec sha256sum "{}" \; | LC_ALL=C sort'
        $actualChecksums = @(Invoke-Docker -Arguments @(
            "run", "--rm",
            "--mount", "type=volume,src=$restoreVolume,dst=/data,readonly",
            "alpine:3.22", "sh", "-c", $checksumCommand
        ) -Capture | Where-Object { $_ })
        $expectedChecksums = @(Get-Content -LiteralPath (Join-Path $BackupPath "media-files.sha256") | Where-Object { $_ })
        if (($expectedChecksums -join "`n") -ne ($actualChecksums -join "`n")) {
            throw "Restored media file checksums do not match the backup."
        }
    }

    Write-Host "BACKUP_RESTORE_VERIFY_OK: $BackupPath"
    Write-Host "Restored and verified $($expectedTables.Count) database tables in an isolated database."
    if ([bool]$manifest.includes_media) {
        Write-Host "Restored and verified $($expectedChecksums.Count) media files in an isolated Docker volume."
    }
    $verificationSucceeded = $true
}
finally {
    try {
        Invoke-Compose -Arguments @("exec", "-T", "mysql", "rm", "-f", $containerDumpPath)
    }
    catch {
        Write-Warning "Unable to remove the temporary SQL file: $($_.Exception.Message)"
    }
    if ($databaseCreated) {
        try {
            $null = Invoke-RootSQL -Statement "DROP DATABASE IF EXISTS ``$restoreDatabase``"
        }
        catch {
            Write-Warning "Unable to remove temporary database '$restoreDatabase': $($_.Exception.Message)"
        }
    }
    if ($volumeCreated) {
        try {
            Invoke-Docker -Arguments @("volume", "rm", $restoreVolume)
        }
        catch {
            Write-Warning "Unable to remove temporary volume '$restoreVolume': $($_.Exception.Message)"
        }
    }
    Pop-Location
}

if ($DeleteBackupAfterVerification -and $verificationSucceeded) {
    $resolvedTemp = (Resolve-Path -LiteralPath $env:TEMP).Path.TrimEnd("\")
    $resolvedBackup = (Resolve-Path -LiteralPath $BackupPath).Path
    $tempPrefix = $resolvedTemp + "\"
    if (-not $resolvedBackup.StartsWith($tempPrefix, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Automatic cleanup is restricted to backups beneath the system temporary directory."
    }
    Remove-Item -LiteralPath $resolvedBackup -Recurse -Force
    Write-Host "REHEARSAL_BACKUP_CLEAN: $resolvedBackup"
}
