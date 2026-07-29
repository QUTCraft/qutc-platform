[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$DumpFile,

    [string]$MediaFile = "",

    [string]$ComposeEnvPath = ""
)

<#
.SYNOPSIS
    从 backup.ps1 生成的备份文件恢复 MySQL 数据库或媒体卷。

.DESCRIPTION
    读取 Compose .env 中的数据库凭据，将 gzip 压缩的 MySQL dump 导入容器内
    的 MySQL 实例；可选恢复媒体卷。恢复前会先暂停 API 容器以避免写冲突，
    完成后重新启动。

.PARAMETER DumpFile
    backup.ps1 生成的 .sql.gz 文件路径（必填）。

.PARAMETER MediaFile
    backup.ps1 生成的 media-*.tar.gz 文件路径（可选）。

.EXAMPLE
    .\scripts\restore.ps1 -DumpFile "backups\20260729-120000\mysql-20260729-120000.sql.gz"

.EXAMPLE
    .\scripts\restore.ps1 -DumpFile "backups\db.sql.gz" -MediaFile "backups\media.tar.gz"
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

# ---- 校验输入文件 --------------------------------------------------------
if (-not (Test-Path -LiteralPath $DumpFile)) {
    throw "Dump file not found: $DumpFile"
}
if ($MediaFile -and -not (Test-Path -LiteralPath $MediaFile)) {
    throw "Media file not found: $MediaFile"
}

$composeFile = Join-Path $repoRootPath "deploy\compose\docker-compose.yml"
$composeArgs = @("-f", $composeFile)

# ---- 暂停 API 避免写冲突 ------------------------------------------------
Write-Host "暂停 API 容器 ..."
& docker compose @composeArgs stop api 2>$null
Write-Host "  API 已暂停"

# ---- MySQL 恢复 ----------------------------------------------------------
$mysqlDatabase = Get-Setting "MYSQL_DATABASE" "qutcraft"
$mysqlUser     = Get-Setting "MYSQL_USER" "qutcraft"
$mysqlPassword = Get-Setting "MYSQL_PASSWORD" "qutcraft"
$tempContainerDump = "/tmp/qutcraft-restore.sql.gz"
$dumpAbsPath = (Resolve-Path $DumpFile).Path
$dumpDir = Split-Path $dumpAbsPath -Parent
$dumpName = Split-Path $dumpAbsPath -Leaf

Write-Host "正在恢复 MySQL ($mysqlDatabase) ..."
# 把 dump 文件复制进 MySQL 容器，再 gunzip | mysql
& docker compose @composeArgs cp $dumpAbsPath "mysql:$tempContainerDump"
if ($LASTEXITCODE -ne 0) {
    throw "failed to copy dump into MySQL container"
}
& docker compose @composeArgs exec -T -e "MYSQL_PWD=$mysqlPassword" mysql sh -c `
    "gunzip -c $tempContainerDump | mysql -u'$mysqlUser' '$mysqlDatabase'"
if ($LASTEXITCODE -ne 0) {
    & docker compose @composeArgs exec -T mysql rm -f $tempContainerDump 2>$null
    throw "mysql restore failed with exit code $LASTEXITCODE"
}
& docker compose @composeArgs exec -T mysql rm -f $tempContainerDump 2>$null
Write-Host "  MySQL 恢复完成"

# ---- 媒体卷恢复 ----------------------------------------------------------
if ($MediaFile) {
    $mediaAbsPath = (Resolve-Path $MediaFile).Path
    $mediaDir = Split-Path $mediaAbsPath -Parent
    $mediaName = Split-Path $mediaAbsPath -Leaf
    $mediaVolume = "qutcraft-platform_media_data"
    Write-Host "正在恢复 media_data 卷 ..."
    & docker run --rm `
        -v "${mediaVolume}:/data" `
        -v "${mediaDir}:/backup:ro" `
        alpine:3.22 sh -c "rm -rf /data/* && tar xzf /backup/$mediaName -C /data"
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "media_data 卷恢复失败 (exit=$LASTEXITCODE)"
    } else {
        Write-Host "  媒体恢复完成"
    }
}

# ---- 重新启动 API --------------------------------------------------------
Write-Host "重新启动 API ..."
& docker compose @composeArgs start api 2>$null
$apiPort = if ($settings.ContainsKey("API_PORT")) { $settings["API_PORT"] } else { "8080" }
Write-Host ""
Write-Host "=== 恢复完成 ==="
Write-Host "验证：docker compose -f `"$composeFile`" ps"
Write-Host "       curl http://localhost:$apiPort/healthz"
