#!/usr/bin/env sh
set -eu

: "${TELEGRAM_BOT_TOKEN:?TELEGRAM_BOT_TOKEN is required}"
: "${TELEGRAM_WEBHOOK_SECRET:?TELEGRAM_WEBHOOK_SECRET is required}"
: "${PUBLIC_URL:?PUBLIC_URL is required}"

case "$PUBLIC_URL" in
  https://*) ;;
  *) echo "PUBLIC_URL must be an https URL" >&2; exit 1 ;;
esac

curl --fail-with-body --silent --show-error \
  -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/setWebhook" \
  -H "Content-Type: application/json" \
  --data "{\"url\":\"${PUBLIC_URL%/}/api/v1/telegram/webhook\",\"secret_token\":\"${TELEGRAM_WEBHOOK_SECRET}\",\"allowed_updates\":[\"message\",\"my_chat_member\"],\"drop_pending_updates\":false}"
echo
