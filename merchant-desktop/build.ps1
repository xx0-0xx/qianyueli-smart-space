param(
    [string]$AdminDir = 'admin-merchant-v3',
    [string]$OutFile = '阡月里商家中心.exe'
)

$ErrorActionPreference = 'Stop'

$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$admin = Join-Path (Split-Path -Parent $here) $AdminDir
$dist = Join-Path $admin 'dist'
$web = Join-Path $here 'web'

Write-Host '== 1/4 构建前端 =='
Push-Location $admin
pnpm build:desktop
Pop-Location
if (-not (Test-Path $dist)) { throw "前端构建产物不存在: $dist" }

Write-Host '== 2/4 拷贝静态资源到 web/ =='
if (Test-Path $web) { Remove-Item $web -Recurse -Force }
Copy-Item $dist $web -Recurse

Write-Host '== 3/4 生成图标资源 =='
$rsrc = Join-Path (go env GOPATH) 'bin\rsrc.exe'
if (Test-Path $rsrc) {
  & $rsrc -arch amd64 -ico (Join-Path $here 'icon.ico') -o (Join-Path $here 'rsrc.syso')
} else {
  Write-Warning '未找到 rsrc，跳过 exe 图标资源（托盘图标不受影响）。安装: go install github.com/akavel/rsrc@latest'
}

Write-Host '== 4/4 编译 exe（无控制台窗口）=='
$out = Join-Path $here $OutFile
Push-Location $here
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
$env:CGO_ENABLED = '0'
go build -ldflags '-s -w -H=windowsgui' -o $out .
Pop-Location

Write-Host ''
Write-Host "完成 -> $out"
