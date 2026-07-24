$ErrorActionPreference = 'Stop'

if (-not $env:TELEGRAM_BOT_TOKEN) {
    throw 'TELEGRAM_BOT_TOKEN is required'
}
if (-not $env:TELEGRAM_WEBHOOK_SECRET) {
    throw 'TELEGRAM_WEBHOOK_SECRET is required'
}
if (-not $env:PUBLIC_URL -or -not $env:PUBLIC_URL.StartsWith('https://')) {
    throw 'PUBLIC_URL must be an https URL'
}

$uri = "https://api.telegram.org/bot$($env:TELEGRAM_BOT_TOKEN)/setWebhook"
$body = @{
    url = "$($env:PUBLIC_URL.TrimEnd('/'))/api/v1/telegram/webhook"
    secret_token = $env:TELEGRAM_WEBHOOK_SECRET
    allowed_updates = @('message', 'my_chat_member')
    drop_pending_updates = $false
} | ConvertTo-Json

$response = Invoke-RestMethod -Method Post -Uri $uri -ContentType 'application/json' -Body $body
if (-not $response.ok) {
    throw "Telegram rejected webhook: $($response.description)"
}
Write-Output "Webhook configured: $($response.description)"
