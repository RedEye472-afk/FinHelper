# FinHelper-Frontend — Деплой

## ✅ Статус: РАЗВЁРНУТО И РАБОТАЕТ

### 🌐 Публичная ссылка
- **Production URL:** https://finhelper-frontend.vercel.app
- **Inspect:** https://vercel.com/fin-helper/finhelper-frontend/FQP4fgxo52znwm4oi3Sxt8WrHziB
- **Alias:** https://finhelper-frontend.vercel.app
- **Готовность:** 27s

### 📦 Платформа
- Vercel (постоянный хостинг, auto-SSL, CDN)
- Проект: finhelper-frontend
- Team: fin-helper

### 🐳 Docker (локальный резерв)
- Собран через `docker compose up -d --build`
- Контейнер: finhelper-frontend-finhelper-frontend-1 (running)
- Порт: 0.0.0.0:5173:80
- Health: http://localhost:5173/health → ok
- Dockerfile: nginx:alpine + готовый dist/ (без npm ci внутри Docker)
- docker-compose.yml, nginx.conf, .dockerignore — в корне проекта

### ⚠️ Cloudflare Tunnel — НЕ работает
- Протокол QUIC: `timeout: no recent network activity` (UDP режется провайдером)
- Протокол HTTP/2: `context canceled` (соединение рвётся через ~40с)
- Причина: сеть/провайдер блокирует long-lived соединения к Cloudflare edge
- Решение: используйте Vercel (работает через HTTPS/CDN, не зависит от туннелей)

### 🔧 Технические детали
| Параметр | Значение |
|----------|----------|
| **ОС** | Windows 10 (build 26200) |
| **Docker** | 29.6.1 |
| **Node.js** | v24.17.0 |
| **npm** | 11.13.0 |
| **Vercel CLI** | 54.20.1 |
| **Vite** | 8.1.3 |
| **React** | 19.2.7 |

### 🚀 Команды
```bash
# Переразвернуть на Vercel (после изменений)
cd C:\Users\user\ZCodeProject\FinHelper-Frontend
npm run build
npx vercel deploy --prod --yes

# Локальный Docker (резерв)
docker compose up -d --build
curl http://localhost:5173/health
```