param(
    [switch]$Fix,
    [switch]$Docker,
    [switch]$Security
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$RepoRoot = Split-Path -Parent $PSScriptRoot
$GoTargets = @(
    "gen/go",
    "services/catalog-service",
    "services/pricing-service"
)
$ServiceTargets = @(
    "services/catalog-service",
    "services/pricing-service"
)
$DockerTargets = @(
    @{
        Name = "catalog-service"
        File = "services/catalog-service/Dockerfile"
        Tag  = "grpc-duo/catalog-service:ci"
    },
    @{
        Name = "pricing-service"
        File = "services/pricing-service/Dockerfile"
        Tag  = "grpc-duo/pricing-service:ci"
    }
)

function Invoke-Step {
    param(
        [string]$Name,
        [scriptblock]$Command
    )

    Write-Host ""
    Write-Host "==> $Name"
    & $Command
    if ($LASTEXITCODE -ne 0) {
        throw "$Name failed with exit code $LASTEXITCODE"
    }
}

Push-Location $RepoRoot
try {
    Invoke-Step "gofmt" {
        $formatTargets = @("gen/go", "services/catalog-service", "services/pricing-service")
        if ($Fix) {
            gofmt -w $formatTargets
            return
        }

        $unformatted = gofmt -l $formatTargets
        if ($unformatted) {
            Write-Host "The following Go files need formatting:"
            $unformatted | ForEach-Object { Write-Host $_ }
            throw "run ./scripts/quality.ps1 -Fix to format Go files"
        }
    }

    foreach ($target in $GoTargets) {
        Invoke-Step "go mod verify ($target)" {
            Push-Location $target
            try {
                go mod verify
            } finally {
                Pop-Location
            }
        }
    }

    foreach ($target in $GoTargets) {
        Invoke-Step "go test ($target)" {
            Push-Location $target
            try {
                go test ./...
            } finally {
                Pop-Location
            }
        }
    }

    foreach ($target in $ServiceTargets) {
        Invoke-Step "go vet ($target)" {
            Push-Location $target
            try {
                go vet ./...
            } finally {
                Pop-Location
            }
        }
    }

    foreach ($target in $ServiceTargets) {
        Invoke-Step "go build ($target)" {
            Push-Location $target
            try {
                go build ./cmd/...
            } finally {
                Pop-Location
            }
        }
    }

    if ($Docker) {
        foreach ($target in $DockerTargets) {
            Invoke-Step "docker build ($($target.Name))" {
                docker build -f $target.File -t $target.Tag .
            }
        }
    }

    if ($Security) {
        foreach ($target in $ServiceTargets) {
            Invoke-Step "govulncheck ($target)" {
                Push-Location $target
                try {
                    go run golang.org/x/vuln/cmd/govulncheck@latest ./...
                } finally {
                    Pop-Location
                }
            }
        }
    }
} finally {
    Pop-Location
}
