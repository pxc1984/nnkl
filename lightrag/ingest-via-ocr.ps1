<#
.SYNOPSIS
    Конвертирует PDF через OCR-сервис в LaTeX и кладёт результат в inputs/ для LightRAG.

.DESCRIPTION
    Пайплайн:
      documents/*.pdf  →  OCR API (Docker :8000)  →  inputs/doc_NNN.tex  →  POST /documents/scan

.PARAMETER SourceDir
    Каталог с исходными PDF (по умолчанию documents/).

.PARAMETER OcrApi
    URL OCR API (по умолчанию http://localhost:8000).

.PARAMETER LightRagApi
    URL LightRAG API (по умолчанию http://127.0.0.1:9621).

.PARAMETER OcrPdfDir
    Каталог PDF, смонтированный в OCR-контейнер как /data/pdfs (по умолчанию ../ocr/data/pdfs).

.PARAMETER SkipScan
    Не вызывать /documents/scan после конвертации.

.PARAMETER File
    Обработать только один PDF (имя файла в SourceDir).
#>
[CmdletBinding()]
param(
    [string]$SourceDir = "",
    [string]$OcrApi = "http://localhost:8000",
    [string]$LightRagApi = "http://127.0.0.1:9621",
    [string]$OcrPdfDir = "",
    [switch]$SkipScan,
    [string]$File = ""
)

$ErrorActionPreference = "Stop"

$root = $PSScriptRoot
if (-not $SourceDir) { $SourceDir = Join-Path $root "documents" }
if (-not $OcrPdfDir) { $OcrPdfDir = (Resolve-Path (Join-Path $root "..\ocr\data\pdfs")).Path }
$inputDir = Join-Path $root "inputs"
$mappingPath = Join-Path $inputDir "ocr_mapping.json"

New-Item -ItemType Directory -Force -Path $inputDir, $OcrPdfDir | Out-Null

function Test-OcrHealth {
    try {
        $health = Invoke-RestMethod -Uri "$OcrApi/api/v1/health" -TimeoutSec 5
        return $health.status -in @("ok", "degraded")
    } catch {
        return $false
    }
}

function Wait-OcrTask {
    param([string]$TaskId, [int]$TimeoutSec = 3600)
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        $status = Invoke-RestMethod -Uri "$OcrApi/api/v1/status/$TaskId"
        Write-Host ("  [{0}] progress={1}% stage={2}" -f $status.status, $status.progress, $status.stage)
        if ($status.status -eq "completed") { return $status }
        if ($status.status -eq "failed") {
            throw "OCR task $TaskId failed: $($status.error)"
        }
        Start-Sleep -Seconds 5
    }
    throw "OCR task $TaskId timed out after ${TimeoutSec}s"
}

function Copy-OcrResult {
    param([string]$TaskId, [string]$DestTex, [string]$SourceName)
    $resultsRoot = Join-Path (Split-Path $OcrPdfDir -Parent) "results"
    $metaPath = Join-Path $resultsRoot "$TaskId\meta.json"
    if (-not (Test-Path $metaPath)) {
        throw "OCR meta not found: $metaPath"
    }
    $meta = Get-Content $metaPath -Raw -Encoding UTF8 | ConvertFrom-Json
    $texPath = $meta.result_path
    if (-not (Test-Path $texPath)) {
        throw "OCR result not found: $texPath"
    }
    Copy-Item -LiteralPath $texPath -Destination $DestTex -Force
    return @{
        source = $SourceName
        task_id = $TaskId
        tex = (Split-Path $DestTex -Leaf)
        ocr_result = $texPath
    }
}

if (-not (Test-OcrHealth)) {
    throw "OCR API unavailable at $OcrApi. Start: cd ocr; docker compose up -d"
}

$pdfFiles = if ($File) {
    @(Get-ChildItem -LiteralPath $SourceDir -Filter $File -File)
} else {
    @(Get-ChildItem -LiteralPath $SourceDir -Filter "*.pdf" -File | Sort-Object Name)
}

if ($pdfFiles.Count -eq 0) {
    throw "No PDF files found in $SourceDir"
}

Write-Host "Converting $($pdfFiles.Count) PDF(s) via OCR..."

$mapping = @()
$index = 1

foreach ($pdf in $pdfFiles) {
    $asciiName = "doc_{0:D3}.pdf" -f $index
    $ocrPdfPath = Join-Path $OcrPdfDir $asciiName
    $destTex = Join-Path $inputDir ("doc_{0:D3}.tex" -f $index)

    Write-Host ""
    Write-Host "[$index/$($pdfFiles.Count)] $($pdf.Name)"

    Copy-Item -LiteralPath $pdf.FullName -Destination $ocrPdfPath -Force

    $body = @{
        file_path = "/data/pdfs/$asciiName"
        output_format = "latex"
        language = "auto"
    } | ConvertTo-Json

    $queued = Invoke-RestMethod -Method Post -Uri "$OcrApi/api/v1/convert" `
        -ContentType "application/json" -Body $body

    Write-Host "  task_id=$($queued.task_id)"
    $final = Wait-OcrTask -TaskId $queued.task_id
    $entry = Copy-OcrResult -TaskId $queued.task_id -DestTex $destTex -SourceName $pdf.Name
    $mapping += $entry
    Write-Host "  -> $destTex"

    $index++
}

$mapping | ConvertTo-Json -Depth 4 | Set-Content -Path $mappingPath -Encoding UTF8
Write-Host ""
Write-Host "Saved mapping: $mappingPath"
Write-Host "Prepared $($mapping.Count) LaTeX file(s) in $inputDir"

if (-not $SkipScan) {
    Write-Host ""
    Write-Host "Triggering LightRAG scan..."
    try {
        $scan = Invoke-RestMethod -Method Post -Uri "$LightRagApi/documents/scan" -TimeoutSec 30
        $scan | ConvertTo-Json -Depth 5
        Write-Host "LightRAG UI: $LightRagApi"
    } catch {
        Write-Warning "LightRAG scan failed (is server running?): $_"
        Write-Host "Start LightRAG: cd lightrag; .\start.ps1"
        Write-Host "Then run: Invoke-RestMethod -Method Post -Uri '$LightRagApi/documents/scan'"
    }
}
