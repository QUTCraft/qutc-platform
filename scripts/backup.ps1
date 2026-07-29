[CmdletBinding()]
param(
    [string]$ComposeEnvPath = "",
    [string]$BackupDir = ""
)

<#
.SYNOPSIS
    备份 QUTCraft Commons 的 MySQL 数据库和媒体卷。

.DESCRIPTION
    从 Compose .env 读取数据库凭据，通过 mysqldump 在 MySQL 容器内导出并压缩，
    同时将 media_data 卷打包为 tar.gz。所有输出写入带时间戳的备份目录。

.EXAMPLE
    .\scripts\backup.ps1
    使用默认 .env 和当前目录创建备份。

.EXAMPLE
    .\scripts\backup.ps1 -BackupDir "D:\backups"
    备份到指定目录。
#>

$ErrorActionPreference = "Stop"
$repoRootPath = Split-Path -Parent $PSScriptRoot

# ---- 解析 Compose .env --------------------------------------------------
if ([string]::IsNullOrWhiteSpace($ComposeEnvPath)) {
    $ComposeEnvPath = Join-Path $repoRootPath "deploy\compose\.env"
}
if (-not (Test-Path -LiteralPath $ComposeEnvPath)) {
    $ComposeEnvPath = Join-Path $repoRootPath "deploy\compose\.env.example"
}
if (-not (Test-Path -LiteralPath $ComposeEnvPath)) {
    throw "Compose environment file was not found at $ComposeEnvPath"
}
$settings = @{}
foreach ($rawLine in Get-Content -LiteralPath $ComposeEnvPath) {
    $line = $rawLine.Trim()
    if ($line.Length -eq 0 -or $line.StartsWith("#")) { continue }
    $separator = $line.IndexOf("=")
    if ($separator -lt 1) { continue }
    $settings[$line.Substring(0, $separator)] = $line.Substring($separator + 1)
}
function Get-Setting {
    param([string]$Name, [string]$Fallback)
    if ($settings.ContainsKey($Name) -and -not [string]::IsNullOrWhiteSpace($settings[$Name])) {
        return $settings[$Name]
    }
    return $Fallback
}

# ---- 备份目录 ------------------------------------------------------------
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
if ([string]::IsNullOrWhiteSpace($BackupDir)) {
    $BackupDir = Join-Path $repoRootPath "backups\$timestamp"
}
New-Item -ItemType Directory -Force -Path $BackupDir | Out-Null
Write-Host "备份目录：$BackupDir"

$composeFile = Join-Path $repoRootPath "deploy\compose\docker-compose.yml"
$composeArgs = @("-f", $composeFile)

# ---- MySQL dump（在容器内压缩，再 cp 出来）-------------------------------
$mysqlDatabase = Get-Setting "MYSQL_DATABASE" "qutcraft"
$mysqlUser     = Get-Setting "MYSQL_USER" "qutcraft"
$mysqlPassword = Get-Setting "MYSQL_PASSWORD" "qutcraft"
$dumpFile = Join-Path $BackupDir "mysql-$timestamp.sql.gz"
$containerDump = "/tmp/qutcraft-dump-$timestamp.sql.gz"

Write-Host "正在导出 MySQL ($mysqlDatabase) ..."
& docker compose @composeArgs exec -T -e "MYSQL_PWD=$mysqlPassword" mysql sh -c `
    "mysqldump -u'$mysqlUser' --single-transaction --routines --triggers --databases '$mysqlDatabase' | gzip > $containerDump"
if ($LASTEXITCODE -ne 0) {
    throw "mysqldump failed with exit code $LASTEXITCODE"
}
& docker compose @composeArgs cp "mysql:$containerDump" $dumpFile
if ($LASTEXITCODE -ne 0) {
    throw "failed to copy dump from MySQL container"
}
& docker compose @composeArgs exec -T mysql rm -f $containerDump 2>$null
$dumpSize = "{0:N0}" -f (Get-Item $dumpFile).Length
Write-Host "  MySQL dump 完成：$dumpFile ($dumpSize 字节)"

# ---- Media 卷打包 --------------------------------------------------------
$mediaFile = Join-Path $BackupDir "media-$timestamp.tar.gz"
Write-Host "正在打包 media_data 卷 ..."
$mediaVolume = "qutcraft-platform_media_data"
& docker run --rm `
    -v "${mediaVolume}:/data:ro" `
    -v "${BackupDir}:/backup" `
    alpine:3.22 tar czf "/backup/media-$timestamp.tar.gz" -C /data .
if ($LASTEXITCODE -ne 0) {
    Write-Warning "media_data 卷备份失败 (exit=$LASTEXITCODE)，可能卷为空或不存在。"
} else {
    $mediaSize = "{0:N0}" -f (Get-Item $mediaFile).Length
    Write-Host "  媒体备份完成：$mediaFile ($mediaSize 字节)"
}

Write-Host ""
Write-Host "=== 备份完成 ==="
Write-Host "恢复方法见：scripts\restore.ps1"
Write-Host "  MySQL: .\scripts\restore.ps1 -DumpFile `"$dumpFile`""
Write-Host "  媒体:  .\scripts\restore.ps1 -MediaFile `"$mediaFile`""
