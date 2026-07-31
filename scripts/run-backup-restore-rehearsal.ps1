[CmdletBinding()]
param(
    [string]$ComposeEnvPath = ""
)

$ErrorActionPreference = "Stop"
$repoRootPath = Split-Path -Parent $PSScriptRoot
$temporaryRoot = Join-Path ([IO.Path]::GetTempPath()) ("qutcraft-backup-gate-" + [Guid]::NewGuid().ToString("N"))

try {
    & (Join-Path $PSScriptRoot "backup-compose.ps1") -ComposeEnvPath $ComposeEnvPath -OutputRoot $temporaryRoot
    if ($LASTEXITCODE -ne 0) {
        throw "Backup creation failed with exit code $LASTEXITCODE."
    }
    $backup = Get-ChildItem -LiteralPath $temporaryRoot -Directory | Select-Object -First 1
    if ($null -eq $backup) {
        throw "Backup creation did not produce an artifact directory."
    }
    & (Join-Path $PSScriptRoot "verify-backup-restore.ps1") -BackupPath $backup.FullName -ComposeEnvPath $ComposeEnvPath -DeleteBackupAfterVerification
    if ($LASTEXITCODE -ne 0) {
        throw "Backup restore verification failed with exit code $LASTEXITCODE."
    }
    Write-Host "BACKUP_RESTORE_REHEARSAL_OK"
}
finally {
    if (Test-Path -LiteralPath $temporaryRoot -PathType Container) {
        $children = @(Get-ChildItem -LiteralPath $temporaryRoot -Force)
        if ($children.Count -eq 0) {
            Remove-Item -LiteralPath $temporaryRoot
        }
        else {
            Write-Warning "Rehearsal artifacts were retained for inspection: $temporaryRoot"
        }
    }
}
