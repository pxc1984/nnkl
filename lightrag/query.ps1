<#
.SYNOPSIS
    Задать вопрос LightRAG по проиндексированным документам.

.PARAMETER Query
    Текст вопроса на русском или английском.

.PARAMETER Mode
    Режим запроса LightRAG: naive, local, global, hybrid (по умолчанию hybrid).
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Query,

    [string]$LightRagApi = "http://127.0.0.1:19621",
    [ValidateSet("naive", "local", "global", "hybrid")]
    [string]$Mode = "hybrid"
)

$ErrorActionPreference = "Stop"

$body = @{
    query = $Query
    mode = $Mode
    only_need_context = $false
} | ConvertTo-Json

$response = Invoke-RestMethod -Method Post -Uri "$LightRagApi/query" `
    -ContentType "application/json" -Body $body -TimeoutSec 300

$response | ConvertTo-Json -Depth 8