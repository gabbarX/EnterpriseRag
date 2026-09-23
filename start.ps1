# PowerShell wrapper for start.sh.
# `bash` in PowerShell resolves to WSL, which may have no usable distro, so run
# start.sh with Git Bash explicitly. Arguments are passed through:
#   .\start.ps1 --rebuild | stop | restart | logs | status
$ErrorActionPreference = 'Stop'

$candidates = @(
    "$env:ProgramFiles\Git\bin\bash.exe",
    "${env:ProgramFiles(x86)}\Git\bin\bash.exe",
    "$env:LOCALAPPDATA\Programs\Git\bin\bash.exe"
)
$gitBash = $candidates | Where-Object { $_ -and (Test-Path $_) } | Select-Object -First 1
if (-not $gitBash) {
    Write-Error "Git Bash not found. Install Git for Windows from https://git-scm.com/download/win"
}

& $gitBash "$PSScriptRoot/start.sh" @args
exit $LASTEXITCODE
