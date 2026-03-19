param(
    [string]$BinaryPath = (Join-Path (Get-Location) "crontui.exe"),
    [string]$TaskPath = $env:CRONTUI_WINDOWS_TASK_PATH,
    [string]$BackupDir = $env:CRONTUI_BACKUP_DIR
)

$ErrorActionPreference = "Stop"

if (-not $TaskPath) {
    $TaskPath = "\CronTUI-CI\"
}

if (-not $BackupDir) {
    if ($env:RUNNER_TEMP) {
        $BackupDir = Join-Path $env:RUNNER_TEMP "crontui-windows-smoke-backups"
    } else {
        $BackupDir = Join-Path (Get-Location) ".tmp\windows-smoke-backups"
    }
}

foreach ($cmdletName in @("Register-ScheduledTask", "Get-ScheduledTask", "Unregister-ScheduledTask")) {
    if (-not (Get-Command $cmdletName -ErrorAction SilentlyContinue)) {
        throw "required ScheduledTasks cmdlet is unavailable: $cmdletName"
    }
}

if (-not (Test-Path $BinaryPath)) {
    throw "crontui binary not found: $BinaryPath"
}

$env:CRONTUI_WINDOWS_TASK_PATH = $TaskPath
$env:CRONTUI_BACKUP_DIR = $BackupDir

function Cleanup-Smoke {
    Get-ScheduledTask -TaskPath $TaskPath -ErrorAction SilentlyContinue | ForEach-Object {
        Unregister-ScheduledTask -TaskName $_.TaskName -TaskPath $_.TaskPath -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
    }
    Remove-Item $BackupDir -Recurse -Force -ErrorAction SilentlyContinue
}

function Invoke-Crontui {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    $output = & $BinaryPath @Arguments 2>&1
    return [pscustomobject]@{
        ExitCode = $LASTEXITCODE
        Output   = @($output)
        Text     = @($output) -join "`n"
    }
}

function Assert-Success {
    param(
        [Parameter(Mandatory = $true)]
        [pscustomobject]$Result,
        [Parameter(Mandatory = $true)]
        [string]$Step
    )

    if ($Result.ExitCode -ne 0) {
        throw "$Step failed: $($Result.Text)"
    }
}

Cleanup-Smoke
New-Item -ItemType Directory -Force -Path $BackupDir | Out-Null

try {
    $add = Invoke-Crontui @("add", "0 9 * * 1-5", "Write-Output hello-from-task", "--desc", "weekday hello")
    Assert-Success $add "add"

    $list = Invoke-Crontui @("list", "--json")
    Assert-Success $list "list"
    $jobs = $list.Text | ConvertFrom-Json
    if ($jobs.Count -ne 1 -or $jobs[0].ID -ne 1) {
        throw "unexpected list output: $($list.Text)"
    }

    $task = Get-ScheduledTask -TaskPath $TaskPath -ErrorAction Stop | Select-Object -First 1 TaskName, TaskPath
    if ($task.TaskName -ne "job-1" -or $task.TaskPath -ne $TaskPath) {
        throw "unexpected Task Scheduler registration: $($task | ConvertTo-Json -Compress)"
    }

    $disable = Invoke-Crontui @("disable", "1")
    Assert-Success $disable "disable"

    $enable = Invoke-Crontui @("enable", "1")
    Assert-Success $enable "enable"

    $run = Invoke-Crontui @("run", "1")
    Assert-Success $run "run"
    if ($run.Text -notmatch "hello-from-task") {
        throw "run output missing expected text: $($run.Text)"
    }

    $backup = Invoke-Crontui @("backup")
    Assert-Success $backup "backup"
    $backupPath = ([regex]::Match($backup.Text, "Backup created: (.+)$", [System.Text.RegularExpressions.RegexOptions]::Multiline)).Groups[1].Value.Trim()
    if (-not $backupPath) {
        throw "backup output missing file path: $($backup.Text)"
    }

    $delete = Invoke-Crontui @("delete", "1")
    Assert-Success $delete "delete"

    $restore = Invoke-Crontui @("restore", [System.IO.Path]::GetFileName($backupPath))
    Assert-Success $restore "restore"

    $final = Invoke-Crontui @("list", "--json")
    Assert-Success $final "final list"
    $finalJobs = $final.Text | ConvertFrom-Json
    if ($finalJobs.Count -ne 1 -or $finalJobs[0].ID -ne 1) {
        throw "restore did not recover job #1: $($final.Text)"
    }

    $reboot = Invoke-Crontui @("add", "@reboot", "Write-Output nope")
    if ($reboot.ExitCode -eq 0) {
        throw "expected @reboot add to fail on native Windows"
    }
    if ($reboot.Text -notmatch "@reboot") {
        throw "@reboot failure did not mention the unsupported schedule: $($reboot.Text)"
    }

    [pscustomobject]@{
        TaskPath = $TaskPath
        Backup   = $backupPath
        Run      = $run.Text
    } | ConvertTo-Json -Depth 3
}
finally {
    Cleanup-Smoke
}

exit 0
