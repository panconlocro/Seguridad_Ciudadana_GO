$ErrorActionPreference = "Stop"

$rootDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$exitCode = 0

Push-Location $rootDir
try {
    & go run ./scripts/models @args
    $exitCode = $LASTEXITCODE
} finally {
    Pop-Location
}

exit $exitCode
