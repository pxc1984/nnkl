$ErrorActionPreference = "Stop"

$result = Invoke-RestMethod `
    -Method Post `
    -Uri "http://127.0.0.1:9621/documents/scan"

$result | ConvertTo-Json -Depth 5
Write-Host "Open http://127.0.0.1:9621"
