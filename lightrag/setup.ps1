$ErrorActionPreference = "Stop"

$root = $PSScriptRoot
Set-Location $root

if (-not (Get-Command ollama -ErrorAction SilentlyContinue)) {
    winget install --id Ollama.Ollama --exact --accept-package-agreements --accept-source-agreements --silent
    $env:Path += ";$env:LOCALAPPDATA\Programs\Ollama"
}

if (-not (Test-Path ".venv")) {
    py -3.12 -m venv .venv
}

& ".\.venv\Scripts\python.exe" -m pip install --upgrade pip
& ".\.venv\Scripts\python.exe" -m pip install -r requirements.txt

ollama pull qwen3:4b-instruct
ollama pull bge-m3

if (-not (Test-Path ".env")) {
    Copy-Item ".env.example" ".env"
}

Write-Host ""
Write-Host "Setup complete. Run .\start.ps1"
