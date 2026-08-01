[CmdletBinding()]
param(
    [string]$WebUrl = "http://127.0.0.1:8082",
    [string]$ApiUrl = "http://127.0.0.1:18080"
)

$ErrorActionPreference = "Stop"
$webBase = $WebUrl.TrimEnd("/")
$apiBase = $ApiUrl.TrimEnd("/")
$routes = @(
    "/",
    "/posts",
    "/projects",
    "/resources",
    "/knowledge",
    "/apply",
    "/login",
    "/admin/audit",
    "/admin/ai",
    "/admin/activity-planner"
)

foreach ($route in $routes) {
    $response = Invoke-WebRequest -Uri "$webBase$route" -Method Get -SkipHttpErrorCheck
    if ($response.StatusCode -ne 200) {
        throw "Web route $route returned HTTP $($response.StatusCode)."
    }
    if ($response.Content -notmatch '<div id="app"></div>') {
        throw "Web route $route did not return the Vue application shell."
    }
    if ([string]::IsNullOrWhiteSpace($response.Headers["Content-Security-Policy"])) {
        throw "Web route $route is missing Content-Security-Policy."
    }
}

$missingPortal = Invoke-WebRequest `
    -Uri "$webBase/portals/quality-gate-missing/index.html" `
    -Method Get `
    -SkipHttpErrorCheck
if ($missingPortal.StatusCode -ne 404) {
    throw "Missing custom portal entry must return HTTP 404, got $($missingPortal.StatusCode)."
}
if ([string]::IsNullOrWhiteSpace($missingPortal.Headers["Content-Security-Policy"])) {
    throw "Missing custom portal response is missing Content-Security-Policy."
}

$health = Invoke-RestMethod -Uri "$apiBase/healthz" -Method Get
if ($health.status -ne "ok") {
    throw "API health response is not ok."
}

$readiness = Invoke-RestMethod -Uri "$apiBase/readyz" -Method Get
if ($readiness.status -ne "ready" -or $readiness.checks.mysql -ne "ok" -or $readiness.checks.redis -ne "ok") {
    throw "API readiness response is not ready."
}

Write-Host "ROUTE_SMOKE_OK: $($routes.Count) SPA routes, portal 404, API liveness/readiness"
