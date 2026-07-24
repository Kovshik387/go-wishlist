# Развёртывание WishTrack

Production-адрес: `https://wish.exchangevolute.ru:8443`

Сервер: `144.31.52.140`

Ветка: `master`

## Первый запуск

Подключитесь к серверу:

```bash
ssh root@144.31.52.140
```

Скачайте и запустите installer:

```bash
curl -fsSL \
  https://raw.githubusercontent.com/Kovshik387/go-wishlist/master/deploy/install-from-git.sh \
  -o /tmp/wishtrack-install.sh

sudo bash /tmp/wishtrack-install.sh
```

Installer:

- ждёт появления A-записи `wish.exchangevolute.ru → 144.31.52.140`;
- не трогает Marzban/Xray и занятый им порт `443`;
- устанавливает Docker Engine и Compose на Ubuntu/Debian;
- клонирует публичный репозиторий в `/opt/wishtrack`;
- скрыто запрашивает Telegram bot token;
- генерирует access/webhook secrets;
- запускает app, worker и Caddy;
- получает TLS-сертификат;
- настраивает Telegram webhook и кнопку меню.

Для WishTrack должны быть свободны и доступны извне порты `80/tcp`,
`8443/tcp` и `8443/udp`. Caddy использует `80` для выпуска TLS-сертификата,
а приложение открывается по HTTPS на `8443`.

Токен не сохраняется в shell history и записывается только в
`/opt/wishtrack/.env` с правами `0600`.

## Обновление

После push в `master` повторите те же две команды. Installer выполнит
`git pull --ff-only`, сохранит `.env`, базу SQLite и загруженные изображения.

## Диагностика

```bash
cd /opt/wishtrack
sudo docker compose -f compose.yaml -f compose.marzban.yaml ps
sudo docker compose -f compose.yaml -f compose.marzban.yaml logs -f app worker caddy
```

## Резервная копия

Данные находятся в именованных Docker volumes:

- `wishtrack_wishtrack_data`;
- `wishtrack_wishtrack_media`.

Перед копированием остановите app и worker, чтобы получить согласованный
снимок SQLite.
