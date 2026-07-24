#!/usr/bin/env bash
set -Eeuo pipefail

# Idempotent production installer for WishTrack.
# Run on Ubuntu/Debian as root:
#   curl -fsSL https://raw.githubusercontent.com/Kovshik387/go-wishlist/master/deploy/install-from-git.sh \
#     -o /tmp/wishtrack-install.sh
#   sudo bash /tmp/wishtrack-install.sh
#
# Re-run the same two commands to pull and deploy future updates.

REPO_URL="${REPO_URL:-https://github.com/Kovshik387/go-wishlist.git}"
REPO_BRANCH="${REPO_BRANCH:-master}"
APP_DIR="${APP_DIR:-/opt/wishtrack}"
DOMAIN="${DOMAIN:-wish.exchangevolute.ru}"
EXPECTED_IP="${EXPECTED_IP:-144.31.52.140}"
APP_SHORT_NAME="${APP_SHORT_NAME:-app}"
PUBLIC_PORT="${PUBLIC_PORT:-8443}"
PUBLIC_URL="https://${DOMAIN}:${PUBLIC_PORT}"
COMPOSE_FILES=(-f compose.yaml -f compose.marzban.yaml)

log() {
  printf '\n[wishtrack] %s\n' "$*"
}

fail() {
  printf '\n[wishtrack] ERROR: %s\n' "$*" >&2
  exit 1
}

if [[ "${EUID}" -ne 0 ]]; then
  fail "run through sudo: sudo bash /tmp/wishtrack-install.sh"
fi

install_base_packages() {
  if ! command -v apt-get >/dev/null 2>&1; then
    fail "automatic installation supports Ubuntu and Debian"
  fi
  export DEBIAN_FRONTEND=noninteractive
  apt-get update
  apt-get install -y ca-certificates curl git gnupg jq openssl
}

install_docker() {
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    systemctl enable --now docker >/dev/null 2>&1 || true
    return
  fi

  log "Installing Docker Engine and Compose"
  # shellcheck disable=SC1091
  . /etc/os-release
  case "${ID}" in
    ubuntu|debian) docker_distribution="${ID}" ;;
    *) fail "unsupported distribution: ${ID}" ;;
  esac

  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL "https://download.docker.com/linux/${docker_distribution}/gpg" \
    -o /etc/apt/keyrings/docker.asc
  chmod a+r /etc/apt/keyrings/docker.asc
  architecture="$(dpkg --print-architecture)"
  printf 'deb [arch=%s signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/%s %s stable\n' \
    "${architecture}" "${docker_distribution}" "${VERSION_CODENAME}" \
    > /etc/apt/sources.list.d/docker.list
  apt-get update
  apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  systemctl enable --now docker
}

wait_for_dns() {
  log "Waiting for ${DOMAIN} to resolve to ${EXPECTED_IP}"
  for _ in $(seq 1 120); do
    if getent ahostsv4 "${DOMAIN}" | awk '{print $1}' | grep -Fxq "${EXPECTED_IP}"; then
      log "DNS is ready"
      return
    fi
    sleep 5
  done
  fail "DNS is not ready. Required A record: wish -> ${EXPECTED_IP}"
}

check_ports() {
  if ! command -v ss >/dev/null 2>&1; then
    return
  fi
  for port in 80 "${PUBLIC_PORT}"; do
    if ss -H -ltn "sport = :${port}" | grep -q .; then
      if ! docker ps --format '{{.Names}}' | grep -q '^wishtrack-caddy-1$'; then
        ss -ltnp "sport = :${port}" || true
        fail "port ${port} is occupied by another service"
      fi
    fi
  done
}

configure_firewall() {
  if command -v ufw >/dev/null 2>&1 && ufw status | grep -q '^Status: active'; then
    ufw allow 80/tcp
    ufw allow "${PUBLIC_PORT}/tcp"
    ufw allow "${PUBLIC_PORT}/udp"
  fi
}

checkout_code() {
  if [[ -d "${APP_DIR}/.git" ]]; then
    log "Updating ${REPO_BRANCH} from GitHub"
    git -C "${APP_DIR}" fetch --prune origin
    git -C "${APP_DIR}" checkout "${REPO_BRANCH}"
    git -C "${APP_DIR}" pull --ff-only origin "${REPO_BRANCH}"
  else
    log "Cloning WishTrack from GitHub"
    if [[ -e "${APP_DIR}" ]] && [[ -n "$(find "${APP_DIR}" -mindepth 1 -maxdepth 1 -print -quit 2>/dev/null)" ]]; then
      fail "${APP_DIR} exists and is not an empty Git repository"
    fi
    install -d -m 0750 "${APP_DIR}"
    git clone --branch "${REPO_BRANCH}" --single-branch "${REPO_URL}" "${APP_DIR}"
  fi
}

read_secret_from_tty() {
  local prompt="$1"
  local value=""
  if [[ ! -r /dev/tty ]]; then
    fail "interactive terminal is required for the first deployment"
  fi
  read -r -s -p "${prompt}" value </dev/tty
  printf '\n' >/dev/tty
  printf '%s' "${value}"
}

create_environment() {
  local env_file="${APP_DIR}/.env"
  if [[ -s "${env_file}" ]]; then
    upsert_env_value "${env_file}" "APP_ENV" "production"
    upsert_env_value "${env_file}" "PUBLIC_URL" "${PUBLIC_URL}"
    upsert_env_value "${env_file}" "FRONTEND_ORIGIN" "${PUBLIC_URL}"
    upsert_env_value "${env_file}" "DOMAIN" "${DOMAIN}"
    upsert_env_value "${env_file}" "PUBLIC_PORT" "${PUBLIC_PORT}"
    chmod 0600 "${env_file}"
    log "Keeping existing production secrets and updating the public URL"
    return
  fi

  local bot_token="${TELEGRAM_BOT_TOKEN:-}"
  if [[ -z "${bot_token}" ]]; then
    bot_token="$(read_secret_from_tty 'Telegram bot token: ')"
  fi
  [[ -n "${bot_token}" ]] || fail "Telegram bot token cannot be empty"

  local bot_response
  bot_response="$(curl -fsS "https://api.telegram.org/bot${bot_token}/getMe")" \
    || fail "Telegram rejected the token"
  local bot_username
  bot_username="$(printf '%s' "${bot_response}" | jq -r '.result.username // empty')"
  [[ -n "${bot_username}" ]] || fail "cannot determine Telegram bot username"

  local access_secret webhook_secret
  access_secret="$(openssl rand -hex 48)"
  webhook_secret="$(openssl rand -hex 32)"
  umask 077
  cat > "${env_file}" <<EOF
APP_ENV=production
PUBLIC_URL=${PUBLIC_URL}
FRONTEND_ORIGIN=${PUBLIC_URL}
DOMAIN=${DOMAIN}
PUBLIC_PORT=${PUBLIC_PORT}
ACCESS_TOKEN_SECRET=${access_secret}
TELEGRAM_BOT_TOKEN=${bot_token}
TELEGRAM_BOT_USERNAME=${bot_username}
TELEGRAM_APP_SHORT_NAME=${APP_SHORT_NAME}
TELEGRAM_WEBHOOK_SECRET=${webhook_secret}
BRAND_NAME=WishTrack
BRAND_EMOJI=gift
BRAND_PRIMARY=#315bc7
BRAND_ACCENT=#c45c4f
NOTIFICATION_DIGEST_WINDOW=5m
WORKER_POLL_INTERVAL=2s
EOF
  chmod 0600 "${env_file}"
  log "Production secrets created"
}

upsert_env_value() {
  local env_file="$1"
  local key="$2"
  local value="$3"
  if grep -q "^${key}=" "${env_file}"; then
    sed -i "s|^${key}=.*|${key}=${value}|" "${env_file}"
  else
    printf '%s=%s\n' "${key}" "${value}" >> "${env_file}"
  fi
}

start_services() {
  cd "${APP_DIR}"
  docker compose "${COMPOSE_FILES[@]}" config >/dev/null
  log "Building and starting app, worker and Caddy (Marzban keeps port 443)"
  docker compose "${COMPOSE_FILES[@]}" up --build -d --remove-orphans

  local ready=0
  for _ in $(seq 1 60); do
    if docker compose "${COMPOSE_FILES[@]}" exec -T app \
      wget -qO- http://127.0.0.1:8080/readyz >/dev/null 2>&1; then
      ready=1
      break
    fi
    sleep 5
  done
  if [[ "${ready}" -ne 1 ]]; then
    docker compose "${COMPOSE_FILES[@]}" logs --tail=120 app worker caddy
    fail "WishTrack did not become ready"
  fi
}

wait_for_https() {
  log "Waiting for Caddy to issue the HTTPS certificate"
  for _ in $(seq 1 60); do
    if curl -fsS "${PUBLIC_URL}/readyz" | grep -q '"status":"ready"'; then
      return
    fi
    sleep 5
  done
  (
    cd "${APP_DIR}"
    docker compose "${COMPOSE_FILES[@]}" logs --tail=120 caddy
  )
  fail "HTTPS is not ready; verify DNS and inbound ports 80/${PUBLIC_PORT}"
}

configure_telegram() {
  local env_file="${APP_DIR}/.env"
  local bot_token webhook_secret bot_username
  bot_token="$(sed -n 's/^TELEGRAM_BOT_TOKEN=//p' "${env_file}" | head -n1)"
  webhook_secret="$(sed -n 's/^TELEGRAM_WEBHOOK_SECRET=//p' "${env_file}" | head -n1)"
  bot_username="$(sed -n 's/^TELEGRAM_BOT_USERNAME=//p' "${env_file}" | head -n1)"

  log "Configuring Telegram webhook and menu button"
  curl -fsS -X POST "https://api.telegram.org/bot${bot_token}/setWebhook" \
    --data-urlencode "url=${PUBLIC_URL}/api/v1/telegram/webhook" \
    --data-urlencode "secret_token=${webhook_secret}" \
    --data-urlencode 'allowed_updates=["message","my_chat_member"]' \
    --data-urlencode 'drop_pending_updates=false' \
    | jq -e '.ok == true' >/dev/null

  curl -fsS -X POST "https://api.telegram.org/bot${bot_token}/setChatMenuButton" \
    --data-urlencode "menu_button={\"type\":\"web_app\",\"text\":\"Открыть WishTrack\",\"web_app\":{\"url\":\"${PUBLIC_URL}\"}}" \
    | jq -e '.ok == true' >/dev/null

  printf '\nWishTrack: %s\nBot: https://t.me/%s\n' "${PUBLIC_URL}" "${bot_username}"
}

install_base_packages
install_docker
wait_for_dns
check_ports
configure_firewall
checkout_code
create_environment
start_services
wait_for_https
configure_telegram

log "Deployment completed"
(
  cd "${APP_DIR}"
  docker compose "${COMPOSE_FILES[@]}" ps
)
