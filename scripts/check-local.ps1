$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot

function Invoke-Checked([string]$Program, [string[]]$Arguments) {
    & $Program @Arguments
    if ($LASTEXITCODE -ne 0) { throw "Check failed ($LASTEXITCODE): $Program $Arguments" }
}

Push-Location (Join-Path $repoRoot 'backend')
try {
    Invoke-Checked 'go' @('test', './...')
} finally { Pop-Location }

foreach ($moduleName in @('admin-web-v3', 'admin-merchant-v3')) {
    $modulePath = Join-Path $repoRoot $moduleName
    if (-not (Test-Path -LiteralPath (Join-Path $modulePath 'node_modules/vite/bin/vite.js'))) {
        throw "Install the locked dependencies in $moduleName first. See README.md."
    }
    $businessViews = if ($moduleName -eq 'admin-web-v3') { 'src/views/ssm' } else { 'src/views/merchant' }
    Push-Location $modulePath
    try {
        Write-Host "Checking $moduleName"
        Invoke-Checked 'node' @('--max_old_space_size=8192', 'node_modules/vue-tsc/bin/vue-tsc.js', '--noEmit')
        Invoke-Checked 'node' @('node_modules/eslint/bin/eslint.js', $businessViews, 'src/router', 'src/config/axios')
        Invoke-Checked 'node' @('--max_old_space_size=8192', 'node_modules/vite/bin/vite.js', 'build', '--mode', 'production')
        Invoke-Checked 'node' @('--max_old_space_size=8192', 'node_modules/vite/bin/vite.js', 'build', '--mode', 'desktop', '--outDir', 'dist-desktop')
    } finally { Pop-Location }
}
Write-Host 'All local source checks passed. This does not validate production services or real hardware.'
