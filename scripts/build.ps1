param(
    [string]$Go = 'go',
    [string]$CC = 'gcc',
    [string]$Version = '0.1.0-dev.3'
)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
if ($Version -notmatch '^\d+\.\d+\.\d+(-[a-zA-Z0-9.]+)?$') { throw 'Invalid version' }
$output = Join-Path $root "build\$Version"
$dll = Join-Path $output "codexgate-v$Version.dll"
if (Test-Path -LiteralPath $dll) { throw "Refusing to overwrite: $dll" }
New-Item -ItemType Directory -Force -Path $output | Out-Null
$old = @{}
$vars = @{ GOCACHE=(Join-Path $root '.cache\go-build'); GOPATH=(Join-Path $root '.cache\gopath'); GOTOOLCHAIN='local'; CGO_ENABLED='1'; CC=$CC }
foreach ($key in $vars.Keys) {
    $old[$key] = [Environment]::GetEnvironmentVariable($key, 'Process')
    [Environment]::SetEnvironmentVariable($key, $vars[$key], 'Process')
}
Push-Location $root
try {
    & $Go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'Tests failed' }
    & $Go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'Vet failed' }
    & $Go build -trimpath -ldflags "-X codexgate/internal/plugin.Version=$Version" -buildmode=c-shared -o $dll ./cmd/codexgate
    if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
    Get-FileHash -LiteralPath $dll -Algorithm SHA256 | Format-List
} finally {
    Pop-Location
    foreach ($key in $old.Keys) { [Environment]::SetEnvironmentVariable($key, $old[$key], 'Process') }
}
