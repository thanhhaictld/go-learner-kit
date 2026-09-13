param()

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$composeFiles = @('-f', 'compose.yaml', '-f', 'compose.watch.yaml')

function Import-EnvironmentFile {
    $envFile = Join-Path $root '.env'
    if (-not (Test-Path -LiteralPath $envFile)) { return }
    Get-Content -LiteralPath $envFile | ForEach-Object {
        if ($_ -match '^\s*([^#=\s]+)\s*=\s*(.*?)\s*$') {
            Set-Item -Path "Env:$($matches[1])" -Value $matches[2]
        }
    }
}

function Start-Watcher([string]$name, [string]$filePath, [string[]]$arguments, [string]$workingDirectory, [hashtable]$variables) {
    foreach ($entry in $variables.GetEnumerator()) { Set-Item -Path "Env:$($entry.Key)" -Value $entry.Value }
    Write-Host "Starting $name"
    return Start-Process -FilePath $filePath -ArgumentList $arguments -WorkingDirectory $workingDirectory -NoNewWindow -PassThru
}

Import-EnvironmentFile
$portalNodeModules = Join-Path $root 'apps/saas-admin-portal/node_modules'
if (-not (Test-Path -LiteralPath $portalNodeModules)) {
    Push-Location (Join-Path $root 'apps/saas-admin-portal')
    try { & npm ci } finally { Pop-Location }
}
$bootstrapOrganizationId = $env:BOOTSTRAP_ORGANIZATION_ID ?? '22222222-2222-2222-2222-222222222222'
$bootstrapAdminUserId = $env:BOOTSTRAP_ADMIN_USER_ID ?? '11111111-1111-1111-1111-111111111111'
$provisioningToken = $env:IDENTITY_PROVISIONING_TOKEN ?? 'development-only-change-me'

Push-Location $root
try {
    & docker compose @composeFiles up -d user-db user-migrate identity-db identity-migrate openfga-db openfga-migrate openfga mailpit
    if ($LASTEXITCODE -ne 0) { throw 'Unable to start watch infrastructure.' }
    & docker compose @composeFiles wait user-migrate identity-migrate openfga-migrate
    if ($LASTEXITCODE -ne 0) { throw 'Watch migrations did not complete successfully.' }
    & docker compose @composeFiles up -d --no-deps prometheus grafana
    if ($LASTEXITCODE -ne 0) { throw 'Unable to start monitoring infrastructure.' }

    $processes = @(
        (Start-Watcher 'authz-service' 'go' @('tool', 'air', '-c', '.air.toml') (Join-Path $root 'src/authz-svc') @{
            HTTP_PORT = '8082'; FGA_API_URL = 'http://localhost:8083'; FGA_STORE_NAME = 'go-learner-authz'; BOOTSTRAP_ORGANIZATION_ID = $bootstrapOrganizationId; BOOTSTRAP_ADMIN_USER_ID = $bootstrapAdminUserId; IDENTITY_PROVISIONING_TOKEN = $provisioningToken
        }),
        (Start-Watcher 'user-service' 'go' @('tool', 'air', '-c', '.air.toml') (Join-Path $root 'src/user-svc') @{
            HTTP_PORT = '8080'; DB_HOST = 'localhost'; DB_PORT = '5443'; DB_USER = 'app'; DB_PASSWORD = 'app'; DB_NAME = 'users'; DB_SSLMODE = 'disable'; AUTHZ_SERVICE_URL = 'http://localhost:8082'
        }),
        (Start-Watcher 'identity-service' 'dotnet' @('watch', 'run', '--no-hot-reload', '--urls', 'http://localhost:8081') (Join-Path $root 'src/identity-svc') @{
            ASPNETCORE_ENVIRONMENT = 'Development'; ConnectionStrings__Default = 'Host=localhost;Port=5444;Database=identity;Username=app;Password=app'; Authz__ServiceUrl = 'http://localhost:8082'; Authz__ProvisioningToken = $provisioningToken; Email__Smtp__Host = 'localhost'; Email__Smtp__Port = '1025'; Email__Smtp__EnableSsl = 'false'
        }),
        (Start-Watcher 'saas-admin-bff' 'dotnet' @('watch', 'run', '--no-hot-reload', '--urls', 'http://localhost:3001') (Join-Path $root 'apps/saas-admin-portal/bff-saas-app') @{
            ASPNETCORE_ENVIRONMENT = 'Development'
        }),
        (Start-Watcher 'saas-admin-vite' 'npm' @('run', 'dev') (Join-Path $root 'apps/saas-admin-portal') @{})
    )

    Write-Host "Control plane: http://localhost:3001"
    Write-Host "Mailpit: http://localhost:8025"
    Wait-Process -Id $processes.Id
}
finally {
    if ($processes) {
        $processes | Where-Object { -not $_.HasExited } | Stop-Process -Force
    }
    Pop-Location
}
