<#
.SYNOPSIS
    Импортирует уникальные .tex из ocr/data/results в lightrag/inputs (без дубликатов).

.PARAMETER ResultsDir
    Каталог результатов OCR.

.PARAMETER TriggerScan
    Вызвать POST /documents/scan после копирования.
#>
[CmdletBinding()]
param(
    [string]$ResultsDir = "",
    [string]$LightRagApi = "http://127.0.0.1:9621",
    [switch]$TriggerScan
)

$ErrorActionPreference = "Stop"
$root = $PSScriptRoot
if (-not $ResultsDir) {
    $ResultsDir = (Resolve-Path (Join-Path $root "..\ocr\data\results")).Path
}
$inputDir = Join-Path $root "inputs"
New-Item -ItemType Directory -Force -Path $inputDir | Out-Null

# Очистить inputs (кроме __parsed__)
Get-ChildItem -Path $inputDir -File | Remove-Item -Force

$metas = Get-ChildItem -Path $ResultsDir -Filter "meta.json" -Recurse -File |
    Sort-Object { $_.Directory.Name }

if ($metas.Count -eq 0) {
    throw "No OCR results in $ResultsDir"
}

$seenHashes = @{}
$mapping = @()
$index = 1

foreach ($metaFile in $metas) {
    $meta = Get-Content $metaFile.FullName -Raw -Encoding UTF8 | ConvertFrom-Json
    if ($meta.status -ne "completed") { continue }

    $texPath = Join-Path $metaFile.Directory.FullName "result.tex"
    if (-not (Test-Path $texPath)) { continue }

    $hash = (Get-FileHash -LiteralPath $texPath -Algorithm SHA256).Hash
    if ($seenHashes.ContainsKey($hash)) {
        Write-Host "Skip duplicate: $($meta.file_path) (same as $($seenHashes[$hash]))"
        continue
    }
    $seenHashes[$hash] = $meta.file_path

    $dest = Join-Path $inputDir ("doc_{0:D3}.tex" -f $index)
    Copy-Item -LiteralPath $texPath -Destination $dest -Force
    $mapping += @{
        index = $index
        source_pdf = $meta.file_path
        task_id = $metaFile.Directory.Name
        tex = (Split-Path $dest -Leaf)
    }
    Write-Host "[$index] $($meta.file_path) -> $dest"
    $index++
}

$mappingPath = Join-Path $root "ocr_mapping.json"
$mapping | ConvertTo-Json -Depth 4 | Set-Content -Path $mappingPath -Encoding UTF8
Write-Host "Imported $($mapping.Count) unique file(s) to $inputDir"

if ($TriggerScan) {
    $scan = Invoke-RestMethod -Method Post -Uri "$LightRagApi/documents/scan" -TimeoutSec 30
    $scan | ConvertTo-Json -Depth 5
}
