$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
$examples = @(
    'backend/config/config.example.yaml',
    'admin-web-v3/.env.example',
    'admin-web-v3/.env.development.example',
    'admin-web-v3/.env.production.example',
    'admin-web-v3/.env.desktop.example',
    'admin-merchant-v3/.env.example',
    'admin-merchant-v3/.env.development.example',
    'admin-merchant-v3/.env.production.example',
    'admin-merchant-v3/.env.desktop.example'
)
foreach ($relativeExample in $examples) {
    $sourcePath = Join-Path $repoRoot $relativeExample
    if ($relativeExample.EndsWith('config.example.yaml')) {
        $relativeTarget = $relativeExample.Replace('config.example.yaml', 'config.yaml')
    } else {
        $relativeTarget = $relativeExample.Substring(0, $relativeExample.Length - '.example'.Length)
    }
    $targetPath = Join-Path $repoRoot $relativeTarget
    if (Test-Path -LiteralPath $targetPath) {
        Write-Host "Skipped existing: $relativeTarget"
        continue
    }
    $content = [System.IO.File]::ReadAllText($sourcePath)
    if ($relativeExample.EndsWith('config.example.yaml')) {
        $randomBytes = New-Object byte[] 48
        $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
        try { $rng.GetBytes($randomBytes) } finally { $rng.Dispose() }
        $content = $content.Replace('change-me-in-production', [Convert]::ToBase64String($randomBytes))
    }
    [System.IO.File]::WriteAllText($targetPath, $content, (New-Object System.Text.UTF8Encoding($false)))
    Write-Host "Created local config: $relativeTarget"
}
Write-Host 'Local config ready. See README.md for services and credentials.'
