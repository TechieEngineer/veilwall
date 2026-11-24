$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')
function Invoke-Checked { param([string]$Exe, [string[]]$Arguments) & $Exe @Arguments; if ($LASTEXITCODE -ne 0) { throw "$Exe failed: $LASTEXITCODE" } }
Invoke-Checked go @('mod','tidy')
if (Test-Path web/package-lock.json) { Invoke-Checked npm.cmd @('--prefix','web','ci') } else { Invoke-Checked npm.cmd @('--prefix','web','install') }
Invoke-Checked npm.cmd @('--prefix','web','test')
Invoke-Checked npm.cmd @('--prefix','web','run','build')
Invoke-Checked go @('vet','./...')
Invoke-Checked go @('test','./...')
New-Item -ItemType Directory -Force bin | Out-Null
$env:CGO_ENABLED = '0'
Invoke-Checked go @('build','-trimpath','-o','bin/veilwall.exe','./cmd/veilwall')
Write-Host 'Ready: .\bin\veilwall.exe --demo'
