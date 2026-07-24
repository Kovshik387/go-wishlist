$ErrorActionPreference = 'Stop'

$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$backupRoot = if ($env:BACKUP_DIR) { $env:BACKUP_DIR } else { Join-Path $PSScriptRoot '..\backups' }
New-Item -ItemType Directory -Force -Path $backupRoot | Out-Null
$backupPath = Join-Path $backupRoot "wishtrack-$stamp.tar.gz"

docker compose stop app worker
try {
    docker compose run --rm --no-deps `
        --entrypoint tar `
        -v "${backupRoot}:/backup" `
        app -czf "/backup/$(Split-Path -Leaf $backupPath)" -C /app data media
    Write-Output "Backup created: $backupPath"
}
finally {
    docker compose start app worker
}
