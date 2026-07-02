$ErrorActionPreference = "Stop"

$root = $PSScriptRoot
$sourceDir = Join-Path $root "documents"
$inputDir = Join-Path $root "inputs"

New-Item -ItemType Directory -Force -Path $inputDir | Out-Null

$files = Get-ChildItem -LiteralPath $sourceDir -Filter "*.pdf" -File | Sort-Object Name
$index = 1
foreach ($file in $files) {
    $destination = Join-Path $inputDir ("doc_{0:D3}.pdf" -f $index)
    Copy-Item -LiteralPath $file.FullName -Destination $destination -Force
    $index++
}

Write-Host "Prepared $($files.Count) PDF files in $inputDir"
