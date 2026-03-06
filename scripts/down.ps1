$ErrorActionPreference = "Stop"

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
Push-Location $root
try {
  docker compose --env-file (Join-Path $root ".env.example") down -v
}
finally {
  Pop-Location
}
