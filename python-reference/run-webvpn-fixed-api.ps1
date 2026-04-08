$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$EnvPath = Join-Path $ScriptDir ".env"
$VenvDir = Join-Path $ScriptDir ".venv"
$PythonExe = Join-Path $VenvDir "Scripts\python.exe"

if (-not (Test-Path $EnvPath)) {
    Write-Error "Missing $EnvPath . Copy .env.example to .env first."
}

if (-not (Test-Path $PythonExe)) {
    py -3 -m venv $VenvDir
}

$oldErrorActionPreference = $ErrorActionPreference
$nativePreferenceExists = $null -ne (Get-Variable PSNativeCommandUseErrorActionPreference -ErrorAction SilentlyContinue)
if ($nativePreferenceExists) {
    $oldNativePreference = $PSNativeCommandUseErrorActionPreference
}

$depsReady = $false
try {
    $ErrorActionPreference = "Continue"
    if ($nativePreferenceExists) {
        $PSNativeCommandUseErrorActionPreference = $false
    }
    & $PythonExe -c "import requests; from Crypto.Cipher import AES" *> $null
    $depsReady = ($LASTEXITCODE -eq 0)
}
finally {
    $ErrorActionPreference = $oldErrorActionPreference
    if ($nativePreferenceExists) {
        $PSNativeCommandUseErrorActionPreference = $oldNativePreference
    }
}

if (-not $depsReady) {
    & $PythonExe -m pip install -r (Join-Path $ScriptDir "requirements.txt")
}

& $PythonExe (Join-Path $ScriptDir "webvpn-fixed-api.py") --env $EnvPath @args
exit $LASTEXITCODE
