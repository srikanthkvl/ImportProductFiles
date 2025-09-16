param(
    [ValidateSet('serve','enqueue')]
    [string]$Action = 'serve',

    [string]$ConfigPath = './config.json',
    [string]$ConfigEnv = 'default',
    [string]$Addr = 'http://localhost:8080',

    [string]$CustomerId,
    [ValidateSet('users','organizations','courses')]
    [string]$ProductType,
    [string]$BlobUri
)

# Ensure we run from repo root (one level up from scripts folder)
if ($PSScriptRoot) { Set-Location (Join-Path $PSScriptRoot '.') }

function Require-Tool($name) {
    $n = Get-Command $name -ErrorAction SilentlyContinue
    if (-not $n) { Write-Error "Required tool not found: $name"; exit 1 }
}

switch ($Action) {
    'serve' {
        Require-Tool go

        write-host "current dir: $(Get-Location)" -ForegroundColor Yellow

        if (Test-Path $ConfigPath) {
            $env:CONFIG_PATH = (Resolve-Path $ConfigPath)
        }
        $env:CONFIG_ENV = $ConfigEnv

        Write-Host "Starting REST server with CONFIG_PATH=$($env:CONFIG_PATH) CONFIG_ENV=$($env:CONFIG_ENV)" -ForegroundColor Cyan
        go run ./cmd/rest
    }
    'enqueue' {
        if (-not $CustomerId -or -not $ProductType -or -not $BlobUri) {
            Write-Error "enqueue requires -CustomerId, -ProductType, -BlobUri"
            exit 2
        }
        $payload = @{ customer_id = $CustomerId; product_type = $ProductType; blob_uri = $BlobUri } | ConvertTo-Json -Compress
        try {
            $resp = Invoke-RestMethod -Method Post -Uri ("{0}/enqueue" -f $Addr.TrimEnd('/')) -Body $payload -ContentType 'application/json'
            Write-Host (ConvertTo-Json $resp) -ForegroundColor Green
        } catch {
            Write-Error $_
            exit 3
        }
    }
}

# Example usage:
# 1) Serve (uses config.json):
#   powershell -ExecutionPolicy Bypass -File .\scripts\rest.ps1 -Action serve -ConfigPath .\config.json -ConfigEnv default
# 2) Enqueue a job:
#   powershell -ExecutionPolicy Bypass -File .\scripts\rest.ps1 -Action enqueue -Addr http://localhost:8080 -CustomerId customer1 -ProductType users -BlobUri file:///data/users.csv


