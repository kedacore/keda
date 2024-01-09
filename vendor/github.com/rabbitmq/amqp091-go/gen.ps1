$DebugPreference = 'Continue'
$ErrorActionPreference = 'Stop'

Set-PSDebug -Off
Set-StrictMode -Version 'Latest' -ErrorAction 'Stop' -Verbose

New-Variable -Name curdir  -Option Constant -Value $PSScriptRoot

$specDir = Resolve-Path -LiteralPath (Join-Path -Path $curdir -ChildPath 'spec')
$amqpSpecXml = Resolve-Path -LiteralPath (Join-Path -Path $specDir -ChildPath 'amqp0-9-1.stripped.extended.xml')
$gen = Resolve-Path -LiteralPath (Join-Path -Path $specDir -ChildPath 'gen.go')
$spec091 = Resolve-Path -LiteralPath (Join-Path -Path $curdir -ChildPath 'spec091.go')

Get-Content -LiteralPath $amqpSpecXml | go run $gen | gofmt | Set-Content -Force -Path $spec091
