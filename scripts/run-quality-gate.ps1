param(
    [switch]$Integration,
    [switch]$Runtime,
    [switch]$StorageIntegration,
    [switch]$BackupRestore
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

function Invoke-Checked {
    param(
        [string]$Label,
        [scriptblock]$Command
    )

    Write-Host "==> $Label"
    & $Command
    if ($LASTEXITCODE -ne 0) {
        throw "$Label failed with exit code $LASTEXITCODE"
    }
}

Push-Location $root
try {
    Invoke-Checked "OpenAPI lint" { python scripts/lint-openapi.py }
    Invoke-Checked "Go/OpenAPI route alignment" { python scripts/check-openapi-routes.py }
    Invoke-Checked "Web/OpenAPI request alignment" { python scripts/check-web-api-contract.py }
    Invoke-Checked "Apifox collection alignment" { python scripts/check-apifox-collection.py }
    Invoke-Checked "Repository secret scan" { python scripts/scan-secrets.py }

    Push-Location "apps/api"
    try {
        Invoke-Checked "Go tests" { go test ./... }
    }
    finally {
        Pop-Location
    }

    Push-Location "apps/web"
    try {
        Invoke-Checked "Web typecheck and production build" { pnpm check }
    }
    finally {
        Pop-Location
    }

    if ($Runtime -or $Integration) {
        Invoke-Checked "Compose route smoke" { & scripts/run-route-smoke.ps1 }
    }

    if ($Integration) {
        Invoke-Checked "S1 content integration" { & scripts/run-s1-integration.ps1 }
        Invoke-Checked "S2 collaboration integration" { & scripts/run-s2-integration.ps1 }
        Invoke-Checked "S3 application integration" { & scripts/run-s3-integration.ps1 }
        Invoke-Checked "S4 portal integration" { & scripts/run-s4-integration.ps1 }
        Invoke-Checked "S5 observability integration" { & scripts/run-s5-observability-integration.ps1 }
        Invoke-Checked "S6 AI agent integration" { & scripts/run-s6-agent-integration.ps1 }
    }

    if ($StorageIntegration) {
        Invoke-Checked "S3/MinIO storage integration" { & scripts/run-storage-integration.ps1 }
    }

    if ($BackupRestore) {
        Invoke-Checked "MySQL/media backup restore rehearsal" { & scripts/run-backup-restore-rehearsal.ps1 }
    }

    Write-Host "QUALITY_GATE_OK"
}
finally {
    Pop-Location
}
