$ErrorActionPreference = "Stop"

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$outDir = Join-Path $root "bin\linux-amd64"
New-Item -ItemType Directory -Force -Path $outDir | Out-Null

$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:GOPROXY = "https://goproxy.cn,direct"
$env:GOSUMDB = "off"

$services = @("api-server", "scheduler", "worker", "notifier")
Push-Location $root
try {
  foreach ($service in $services) {
    go build -trimpath -ldflags="-s -w" -o (Join-Path $outDir $service) ".\cmd\$service"
  }
}
finally {
  Pop-Location
}
