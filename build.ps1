# build.ps1 - Build script for the Go CLI application

$BINARY_NAME = "mxlint"
$GOBASE = Get-Location
$GOBIN = Join-Path $GOBASE "bin"
$GOPKG = $GOBASE

# Ensure bin folder exists
if (-not (Test-Path $GOBIN)) {
    New-Item -ItemType Directory -Path $GOBIN | Out-Null
}

function Clean {
    Write-Host "Cleaning..."
    go clean
    Get-ChildItem "$GOBIN\$BINARY_NAME*" -ErrorAction SilentlyContinue | Remove-Item -Force
}

function Test {
    Write-Host "Running tests..."
    go test -v ./...
}

function Deps {
    Write-Host "Fetching dependencies..."
    go mod tidy
}

function Build-MacOS {
    Write-Host "Building for macOS amd64..."
    $env:GOOS="darwin"; $env:GOARCH="amd64"
    go build -o "$GOBIN/$BINARY_NAME-darwin-amd64" $GOPKG
    Remove-Item Env:\GOOS, Env:\GOARCH
}

function Build-MacOS-Arm64 {
    Write-Host "Building for macOS arm64..."
    $env:GOOS="darwin"; $env:GOARCH="arm64"
    go build -o "$GOBIN/$BINARY_NAME-darwin-arm64" $GOPKG
    Remove-Item Env:\GOOS, Env:\GOARCH
}

function Build-Windows {
    Write-Host "Building for Windows amd64..."
    $env:GOOS="windows"; $env:GOARCH="amd64"
    go build -o "$GOBIN/$BINARY_NAME-windows-amd64.exe" $GOPKG
    Remove-Item Env:\GOOS, Env:\GOARCH
}

# Parse args
param (
    [string]$Target = "build-windows"
)

switch ($Target) {
    "clean" { Clean }
    "test" { Test }
    "deps" { Deps }
    "build-macos" { Build-MacOS }
    "build-macos-arm64" { Build-MacOS-Arm64 }
    "build-windows" { Build-Windows }
    "all" {
        Clean
        Deps
        Test
        Build-MacOS
        Build-Windows
        Build-MacOS-Arm64
    }
    default { Write-Host "Unknown target: $Target" }
}
