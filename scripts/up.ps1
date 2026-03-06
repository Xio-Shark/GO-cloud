$ErrorActionPreference = "Stop"

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
Push-Location $root
try {
  & (Join-Path $PSScriptRoot "build-linux.ps1")
  docker compose --env-file (Join-Path $root ".env.example") up -d --build
}
finally {
  Pop-Location
}
