# FinHelper Scripts

Скрипты для профессиональной веб-автоматизации, скрапинга и антидетекта.

## Установка

```bash
# Установить зависимости
cd C:\Users\user\ZCodeProject\FinHelper\scripts
python -m pip install -r requirements.txt
```

## Компоненты

### 1. Stealth Browser (`stealth_browser.py`)
Профессиональный браузер с антидетектом на базе Playwright.

```bash
# Открыть сайт в обычном режиме
python stealth_browser.py --url https://example.com

# Открыть в headless режиме и сделать скриншот
python stealth_browser.py --url https://example.com --headless --screenshot shot.png

# Использовать прокси
python stealth_browser.py --url https://example.com --proxy http://user:pass@1.2.3.4:8080

# Решить капчу
python stealth_browser.py --solve-captcha --url https://example.com
```

### 2. CAPTCHA Solver (`capzy_solver.py`)
Интеграция с сервисом capzy для решения капч.

```python
from capzy_solver import solve_captcha

token = solve_captcha(
    site_url="https://example.com",
    site_key="6LeIxAcTAAAAAJcZVRqyHh71UMIEGNQ_MXjiZKhI"
)
```

### 3. Anti-Bot Tools (`anti_detect.py`)
Набор утилит для обхода детекции ботов:
- Генерация реалистичных User-Agent'ов
- Подмена заголовков
- Эмуляция человеческого поведения

## API Ключи

- **Capzy**: `capzy_e1ccf94c1ba7fe9c48694465991061d7e6150a8971b79ff2`

## Тестирование

```bash
# Запуск всех тестов
pytest test_*.py -v

# Тест конкретного модуля
python -m pytest test_stealth_browser.py -v
```
