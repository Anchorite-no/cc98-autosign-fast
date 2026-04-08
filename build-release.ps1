param(
    [string]$GoBin = $env:GO_BIN
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($GoBin)) {
    $GoBin = "go"
}

$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$Dist = Join-Path $Root "dist"
$Readme = Join-Path $Root "README.md"
$EnvExample = Join-Path $Root ".env.example"

if (Test-Path $Dist) {
    Remove-Item -Recurse -Force $Dist
}
New-Item -ItemType Directory -Path $Dist | Out-Null

& $GoBin version | Out-Null

$windowsFolder = Join-Path $Dist "cc98-autosign-fast-windows-amd64"
New-Item -ItemType Directory -Path $windowsFolder | Out-Null
Copy-Item $EnvExample (Join-Path $windowsFolder ".env")
Copy-Item $Readme (Join-Path $windowsFolder "README.md")
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
& $GoBin build -trimpath -ldflags "-s -w" -o (Join-Path $windowsFolder "cc98-autosign-fast.exe") .\src

$linuxFolder = Join-Path $Dist "cc98-autosign-fast-linux-amd64"
New-Item -ItemType Directory -Path $linuxFolder | Out-Null
Copy-Item $EnvExample (Join-Path $linuxFolder ".env")
Copy-Item $Readme (Join-Path $linuxFolder "README.md")
$env:GOOS = "linux"
$env:GOARCH = "amd64"
& $GoBin build -trimpath -ldflags "-s -w" -o (Join-Path $linuxFolder "cc98-autosign-fast") .\src

$env:CGO_ENABLED = $null
$env:GOOS = $null
$env:GOARCH = $null

$windowsArchive = Join-Path $Dist "cc98-autosign-fast-windows-amd64.zip"
Compress-Archive -Path (Join-Path $windowsFolder "*") -DestinationPath $windowsArchive -Force

$linuxArchive = Join-Path $Dist "cc98-autosign-fast-linux-amd64.tar.gz"
& tar.exe -czf $linuxArchive -C $Dist "cc98-autosign-fast-linux-amd64"

Write-Output "Release artifacts:"
Write-Output " - $windowsArchive"
Write-Output " - $linuxArchive"
