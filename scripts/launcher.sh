#!/usr/bin/env bash
# ==============================================================================
# FinHelper Professional Toolkit — Launcher
# ==============================================================================
# Универсальный лаунчер для скриптов FinHelper.
# Запуск: ./launcher.sh [команда] [опции]
# ==============================================================================

set -euo pipefail

SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PYTHON="python"
CAPZY_KEY="${CAPZY_API_KEY:-capzy_e1ccf94c1ba7fe9c48694465991061d7e6150a8971b79ff2}"

# Цвета
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

show_help() {
    cat << EOF
${BLUE}FinHelper Professional Toolkit${NC}
Использование: ./launcher.sh <команда> [опции]

Команды:
  browser   [--url URL] [--headless] [--proxy P]  Запустить стелс-браузер
  captcha   --sitekey KEY --url URL [--key KEY]    Решить капчу
  scrape    --url URL [--output DIR]               Скрапнуть страницу
  translate --text "..." --to LANG                 Перевести текст
  test                                             Запустить тесты
  bypass    --url URL                              Проверить обход Cloudflare
  ua-test                                          Тест User-Agent'ов
  help                                            Показать помощь

Примеры:
  ./launcher.sh browser --url https://example.com
  ./launcher.sh captcha --sitekey 6Le-wxAaAAAAAEHk1FiVnE1L-HEB6nFaA6pBnF8v --url https://example.com
  ./launcher.sh translate --text "Hello world" --to ru
  ./launcher.sh test
  ./launcher.sh bypass --url https://example.com
EOF
}

# ── Стелс-браузер ──────────────────────────────────────────────────────────
cmd_browser() {
    echo -e "${BLUE}[+] Запуск Stealth Browser...${NC}"
    cd "$SCRIPTS_DIR"
    $PYTHON stealth_browser.py "$@"
}

# ── Капча ───────────────────────────────────────────────────────────────────
cmd_captcha() {
    echo -e "${BLUE}[+] Решение капчи через capzy...${NC}"
    cd "$SCRIPTS_DIR"
    $PYTHON capzy_solver.py --key "$CAPZY_KEY" --solve "$@"
}

# ── Скрапинг ────────────────────────────────────────────────────────────────
cmd_scrape() {
    local url=""
    local output_dir="./downloads"
    
    # Парсим свои аргументы
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --url) url="$2"; shift 2 ;;
            --output) output_dir="$2"; shift 2 ;;
            *) echo "Неизвестный аргумент: $1"; exit 1 ;;
        esac
    done
    
    if [[ -z "$url" ]]; then
        echo -e "${RED}Ошибка: --url обязателен${NC}"
        exit 1
    fi
    
    mkdir -p "$output_dir"
    echo -e "${BLUE}[+] Скрапинг: $url${NC}"
    echo -e "${BLUE}[+] Сохранение в: $output_dir${NC}"
    
    # Используем Python для скрапинга
    $PYTHON -c "
import sys
sys.path.insert(0, '$SCRIPTS_DIR')
from anti_detect import CloudflareBypass

url = '$url'
output = '$output_dir'

scraper = CloudflareBypass.get_cloudscraper()
resp = scraper.get(url, timeout=30)

print(f'Статус: {resp.status_code}')
print(f'Размер: {len(resp.content)} bytes')

# Сохраняем HTML
import hashlib
filename = hashlib.md5(url.encode()).hexdigest()[:12]
with open(f'{output}/{filename}.html', 'w', encoding='utf-8') as f:
    f.write(resp.text)

print(f'HTML сохранён: {output}/{filename}.html')
"
}

# ── Перевод ─────────────────────────────────────────────────────────────────
cmd_translate() {
    local text=""
    local to="ru"
    local from="auto"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --text) text="$2"; shift 2 ;;
            --to) to="$2"; shift 2 ;;
            --from) from="$2"; shift 2 ;;
            *) echo "Неизвестный аргумент: $1"; exit 1 ;;
        esac
    done
    
    if [[ -z "$text" ]]; then
        echo -e "${RED}Ошибка: --text обязателен${NC}"
        exit 1
    fi
    
    echo -e "${BLUE}[+] Перевод: '$text' → $to${NC}"
    $PYTHON -c "
from deep_translator import GoogleTranslator
try:
    result = GoogleTranslator(source='$from', target='$to').translate('$text')
    print(f'Результат: ${GREEN}{result}${NC}')
except Exception as e:
    print(f'Ошибка: {e}')
"
}

# ── Тесты ───────────────────────────────────────────────────────────────────
cmd_test() {
    echo -e "${BLUE}[+] Запуск тестов...${NC}"
    cd "$SCRIPTS_DIR"
    $PYTHON -m pytest test_all.py -v
}

# ── Тест UA ─────────────────────────────────────────────────────────────────
cmd_ua_test() {
    echo -e "${BLUE}[+] Тест генерации User-Agent'ов...${NC}"
    $PYTHON -c "
from anti_detect import UserAgentManager
uam = UserAgentManager()
for browser in ['chrome', 'firefox', 'edge']:
    for _ in range(3):
        ua = uam.get_random(browser)
        print(f'  [${browser}] ${ua}')
    print()
"
}

# ── Обход Cloudflare ────────────────────────────────────────────────────────
cmd_bypass() {
    local url=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --url) url="$2"; shift 2 ;;
            *) echo "Неизвестный аргумент: $1"; exit 1 ;;
        esac
    done
    
    if [[ -z "$url" ]]; then
        echo -e "${RED}Ошибка: --url обязателен${NC}"
        exit 1
    fi
    
    echo -e "${BLUE}[+] Тест обхода Cloudflare: $url${NC}"
    $PYTHON -c "
from anti_detect import CloudflareBypass
scraper = CloudflareBypass.get_cloudscraper()
try:
    resp = scraper.get('$url', timeout=30)
    print(f'Статус: ${resp.status_code}')
    print(f'Заголовки:')
    for k, v in resp.headers.items():
        if k.lower() in ('server', 'cf-ray', 'x-frame-options', 'content-type'):
            print(f'  {k}: {v}')
    print(f'Размер: {len(resp.content)} bytes')
    print(f'${GREEN}✓ Cloudflare обойдён!${NC}')
except Exception as e:
    print(f'${RED}✗ Ошибка: {e}${NC}')
"
}

# ── Главный диспетчер ───────────────────────────────────────────────────────
main() {
    if [[ $# -eq 0 ]]; then
        show_help
        exit 0
    fi
    
    local cmd="$1"
    shift
    
    case "$cmd" in
        browser)    cmd_browser "$@" ;;
        captcha)    cmd_captcha "$@" ;;
        scrape)     cmd_scrape "$@" ;;
        translate)  cmd_translate "$@" ;;
        test)       cmd_test ;;
        ua-test)    cmd_ua_test ;;
        bypass)     cmd_bypass "$@" ;;
        help|--help|-h) show_help ;;
        *)
            echo -e "${RED}Неизвестная команда: $cmd${NC}"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
