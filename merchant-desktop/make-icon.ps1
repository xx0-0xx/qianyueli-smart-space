param(
  [Parameter(Mandatory = $true)]
  [ValidateScript({ Test-Path -LiteralPath $_ -PathType Leaf })]
  [string]$Source,
  [string]$OutFile = 'icon.ico'
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing

$sourceImage = [System.Drawing.Image]::FromFile($Source)
$sizes = @(16, 24, 32, 48, 64, 128, 256)
$images = @()

foreach ($size in $sizes) {
  $bmp = New-Object System.Drawing.Bitmap($size, $size)
  $graphics = [System.Drawing.Graphics]::FromImage($bmp)
  $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
  $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
  $graphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
  $graphics.DrawImage($sourceImage, 0, 0, $size, $size)
  $graphics.Dispose()

  $stream = New-Object System.IO.MemoryStream
  $bmp.Save($stream, [System.Drawing.Imaging.ImageFormat]::Png)
  $bytes = $stream.ToArray()
  $stream.Dispose()
  $bmp.Dispose()

  $images += ,@($size, $bytes)
}

$sourceImage.Dispose()

$file = [System.IO.File]::Create((Join-Path $PSScriptRoot $OutFile))
$writer = New-Object System.IO.BinaryWriter($file)

$writer.Write([UInt16]0)             # reserved
$writer.Write([UInt16]1)             # type: icon
$writer.Write([UInt16]$images.Count) # image count

$offset = 6 + (16 * $images.Count)
foreach ($image in $images) {
  $size = [int]$image[0]
  $data = [byte[]]$image[1]
  $dim = if ($size -ge 256) { 0 } else { $size }
  $writer.Write([Byte]$dim)          # width
  $writer.Write([Byte]$dim)          # height
  $writer.Write([Byte]0)             # color count
  $writer.Write([Byte]0)             # reserved
  $writer.Write([UInt16]1)           # planes
  $writer.Write([UInt16]32)          # bit count
  $writer.Write([UInt32]$data.Length) # bytes
  $writer.Write([UInt32]$offset)     # offset
  $offset += $data.Length
}

foreach ($image in $images) {
  $writer.Write([byte[]]$image[1])
}

$writer.Close()
$file.Close()

Write-Host "icon.ico generated: $((Get-Item (Join-Path $PSScriptRoot $OutFile)).Length) bytes"
