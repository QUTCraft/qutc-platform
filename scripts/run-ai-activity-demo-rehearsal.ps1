[CmdletBinding()]
param(
    [ValidateRange(1, 10)]
    [int]$Rounds = 3,
    [ValidateRange(10, 300)]
    [int]$PollTimeoutSeconds = 120,
    [string]$ApiUrl = "",
    [string]$EnvFile = "",
    [string]$OutputPath = "",
    [string]$SourceQuery = "项目",
    [switch]$AllowMock
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
if ([string]::IsNullOrWhiteSpace($EnvFile)) {
    $EnvFile = Join-Path $root "deploy/compose/.env"
}
if (-not (Test-Path -LiteralPath $EnvFile)) {
    throw "Compose environment file was not found. Create the ignored deploy/compose/.env first."
}

$settings = @{}
foreach ($rawLine in Get-Content -LiteralPath $EnvFile) {
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
    param([string]$Name, [string]$Fallback = "")
    if ($settings.ContainsKey($Name) -and -not [string]::IsNullOrWhiteSpace($settings[$Name])) {
        return $settings[$Name]
    }
    return $Fallback
}

if ([string]::IsNullOrWhiteSpace($ApiUrl)) {
    $ApiUrl = "http://127.0.0.1:$(Get-Setting 'API_PORT' '8080')"
}
$ApiUrl = $ApiUrl.TrimEnd("/")
if ([string]::IsNullOrWhiteSpace($OutputPath)) {
    $stamp = (Get-Date).ToUniversalTime().ToString("yyyyMMdd-HHmmss")
    $OutputPath = Join-Path $root "tmp/agent-rehearsal/activity-planner-$stamp.json"
}

$adminEmail = Get-Setting "BOOTSTRAP_ADMIN_EMAIL"
$adminPassword = Get-Setting "BOOTSTRAP_ADMIN_PASSWORD"
if ([string]::IsNullOrWhiteSpace($adminEmail) -or [string]::IsNullOrWhiteSpace($adminPassword)) {
    throw "BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD are required in the ignored environment file."
}

$webSession = [Microsoft.PowerShell.Commands.WebRequestSession]::new()

function Invoke-QUTCAPI {
    param(
        [ValidateSet("GET", "POST", "PUT")]
        [string]$Method,
        [string]$Path,
        [string]$Token = "",
        [object]$Body = $null,
        [string]$RequestID = ""
    )
    $headers = @{ Accept = "application/json" }
    if (-not [string]::IsNullOrWhiteSpace($Token)) {
        $headers.Authorization = "Bearer $Token"
    }
    if (-not [string]::IsNullOrWhiteSpace($RequestID)) {
        $headers["X-Request-ID"] = $RequestID
    }
    $arguments = @{
        Uri = "$ApiUrl$Path"
        Method = $Method
        Headers = $headers
        TimeoutSec = 15
        WebSession = $webSession
    }
    if ($null -ne $Body) {
        $arguments.ContentType = "application/json"
        $arguments.Body = $Body | ConvertTo-Json -Depth 12 -Compress
    }
    try {
        return Invoke-RestMethod @arguments
    } catch {
        $status = "network"
        if ($null -ne $_.Exception.Response) {
            $status = [int]$_.Exception.Response.StatusCode
        }
        throw "API $Method $Path failed (status=$status)."
    }
}

$token = ""
try {
$login = Invoke-QUTCAPI -Method POST -Path "/api/v1/auth/login" -Body @{
    email = $adminEmail
    password = $adminPassword
}
$token = [string]$login.data.access_token
if ([string]::IsNullOrWhiteSpace($token)) {
    throw "Login succeeded without an access token."
}

$configuration = Invoke-QUTCAPI -Method GET -Path "/api/v1/admin/ai/config" -Token $token
$provider = $configuration.data.provider
if (-not $provider.enabled -or -not $provider.configured) {
    throw "The configured model provider is unavailable."
}
if (-not $AllowMock -and $provider.mode -ne "real") {
    throw "Competition rehearsal requires provider.mode=real. Use -AllowMock only for development diagnostics."
}

$searchQueries = @($SourceQuery, "活动", "组织", "规范", "门户") | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Select-Object -Unique
$source = $null
foreach ($query in $searchQueries) {
    $search = Invoke-QUTCAPI -Method POST -Path "/api/v1/admin/ai/knowledge/search" -Token $token -Body @{ query = $query; limit = 10 }
    if ($search.data.Count -gt 0) {
        $source = $search.data[0]
        break
    }
}
if ($null -eq $source) {
    throw "No current-organization knowledge source matched the rehearsal queries."
}

$rehearsalID = "rehearsal-$([Guid]::NewGuid().ToString('N').Substring(0, 12))"
$results = @()
for ($round = 1; $round -le $Rounds; $round++) {
    $roundResult = [ordered]@{
        round = $round
        succeeded = $false
        status = "not_started"
        duration_ms = 0
        plan_id = ""
        request_id = "$rehearsalID-round-$round-create"
        provider = [string]$provider.provider
        mode = [string]$provider.mode
        model = [string]$provider.model
        prompt_version = ""
        citations = 0
        proposed_actions = 0
        input_tokens = 0
        output_tokens = 0
        failure = ""
    }
    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    try {
        $start = (Get-Date).ToUniversalTime().AddDays(30 + $round)
        $created = Invoke-QUTCAPI -Method POST -Path "/api/v1/admin/ai/activity-plans" -Token $token -RequestID $roundResult.request_id -Body @{
            title = "比赛主演示稳定性演练 $round · $rehearsalID"
            objective = "验证校园组织活动策划服务在真实模型下能够稳定生成带引用、可人工审查且不越权执行的方案"
            audience = "校园社团负责人、活动组织者与参赛评审"
            venue = "校内多功能活动空间"
            starts_at = $start.ToString("yyyy-MM-ddTHH:mm:ssZ")
            ends_at = $start.AddHours(4).ToString("yyyy-MM-ddTHH:mm:ssZ")
            expected_participants = 60
            budget = "1500 元以内"
            constraints = "必须保留人工批准；不得自动发布、审批、发送消息或调用服务器；需要场地、安全和应急预案"
            context_refs = @(@{ type = "content"; id = [string]$source.id })
        }
        $plan = $created.data
        $roundResult.plan_id = [string]$plan.id
        $deadline = (Get-Date).AddSeconds($PollTimeoutSeconds)
        while ((Get-Date) -lt $deadline -and $plan.status -eq "generating") {
            Start-Sleep -Milliseconds 700
            $plan = (Invoke-QUTCAPI -Method GET -Path "/api/v1/admin/ai/activity-plans/$($roundResult.plan_id)" -Token $token).data
        }
        $roundResult.status = [string]$plan.status
        $roundResult.prompt_version = [string]$plan.run.prompt_version
        $roundResult.citations = @($plan.run.citations).Count
        $roundResult.proposed_actions = @($plan.proposed_actions).Count
        $roundResult.input_tokens = [int]$plan.run.input_tokens
        $roundResult.output_tokens = [int]$plan.run.output_tokens
        $markdown = [string]$plan.run.output_markdown
        $citationMarker = "qutc://knowledge/$([string]$source.id)"
        $roundResult.succeeded = $plan.status -eq "ready" `
            -and $plan.run.status -eq "succeeded" `
            -and $plan.run.prompt_version -eq "activity-planner/v2" `
            -and @($plan.run.citations).Count -ge 1 `
            -and @($plan.proposed_actions).Count -eq 6 `
            -and $markdown.TrimStart().StartsWith("# ") `
            -and $markdown.Contains($citationMarker)
        if (-not $roundResult.succeeded) {
            $roundResult.failure = if ($plan.status -eq "generating") { "poll_timeout" } elseif (-not [string]::IsNullOrWhiteSpace([string]$plan.run.failure_code)) { [string]$plan.run.failure_code } else { "contract_check_failed" }
        }
    } catch {
        $roundResult.status = "failed"
        $roundResult.failure = "rehearsal_request_failed"
    } finally {
        $stopwatch.Stop()
        $roundResult.duration_ms = $stopwatch.ElapsedMilliseconds
        $results += [pscustomobject]$roundResult
        $state = if ($roundResult.succeeded) { "PASS" } else { "FAIL" }
        Write-Host ("DEMO_REHEARSAL_ROUND {0}/{1} {2} status={3} latency={4}ms citations={5} actions={6}" -f $round, $Rounds, $state, $roundResult.status, $roundResult.duration_ms, $roundResult.citations, $roundResult.proposed_actions)
    }
}

$passed = @($results | Where-Object succeeded).Count
$durations = @($results | ForEach-Object duration_ms | Sort-Object)
$averageLatency = if ($durations.Count) { [math]::Round(($durations | Measure-Object -Average).Average) } else { 0 }
$report = [ordered]@{
    version = "activity-planner-rehearsal/v1"
    rehearsal_id = $rehearsalID
    generated_at = (Get-Date).ToUniversalTime().ToString("o")
    api_url = $ApiUrl
    provider = @{ provider = $provider.provider; mode = $provider.mode; model = $provider.model }
    prompt_version = "activity-planner/v2"
    rounds_requested = $Rounds
    source_count = 1
    human_evaluation = "pending"
    human_approval = "pending"
    summary = @{
        passed = $passed
        failed = $Rounds - $passed
        average_latency_ms = $averageLatency
        input_tokens = ($results | Measure-Object input_tokens -Sum).Sum
        output_tokens = ($results | Measure-Object output_tokens -Sum).Sum
    }
    rounds = $results
}

$outputDirectory = Split-Path -Parent $OutputPath
New-Item -ItemType Directory -Path $outputDirectory -Force | Out-Null
$report | ConvertTo-Json -Depth 12 | Set-Content -LiteralPath $OutputPath -Encoding utf8
Write-Host "AI_ACTIVITY_DEMO_REHEARSAL: provider=$($provider.provider) mode=$($provider.mode) passed=$passed/$Rounds average_latency=${averageLatency}ms"
Write-Host "AI_ACTIVITY_DEMO_REHEARSAL_REPORT: $OutputPath"
if ($passed -ne $Rounds) {
    throw "Only $passed/$Rounds rehearsal rounds passed."
}
} finally {
    if (-not [string]::IsNullOrWhiteSpace($token)) {
        try {
            $null = Invoke-QUTCAPI -Method POST -Path "/api/v1/auth/logout" -Token $token -Body @{}
        } catch {
            Write-Warning "The rehearsal completed, but the temporary login session could not be revoked immediately."
        }
    }
}
