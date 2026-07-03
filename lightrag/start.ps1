$ErrorActionPreference = "Stop"

$root = $PSScriptRoot
Set-Location $root

if (-not (Test-Path ".env")) {
    Copy-Item ".env.example" ".env"
}

& "$root\prepare-input.ps1"

$env:PYTHONUTF8 = "1"
$env:PYTHONIOENCODING = "utf-8"
$env:INPUT_DIR = (Resolve-Path "$root\inputs").Path

Get-Content "$root\.env" |
    Where-Object { $_ -match "^[A-Za-z_][A-Za-z0-9_]*=" } |
    ForEach-Object {
        $name, $value = $_ -split "=", 2
        Set-Item -Path "Env:$name" -Value $value
    }

& "$root\.venv\Scripts\lightrag-server.exe" `
    --host 127.0.0.1 `
    --port 9621 `
    --working-dir "$root\rag_storage" `
    --input-dir "$root\inputs" `
    --workers 1
