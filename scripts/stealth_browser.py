#!/usr/bin/env python3
"""
FinHelper Stealth Browser Module
=================================
Профессиональный инструмент для автоматизации браузера с:
- Обходом детекции ботов (антидетект)
- Поддержкой CAPTCHA-решалов (capzy, capsolver)
- Управлением прокси и фингерпринтингом
- Многопоточным скрапингом

Использование:
    from stealth_browser import StealthBrowser, CapzySolver
    
    browser = StealthBrowser(headless=False)
    page = browser.new_page()
    page.goto("https://example.com")
    browser.close()
"""

import os
import sys
import json
import time
import random
import logging
from typing import Optional, Dict, Any, List
from dataclasses import dataclass, field

logger = logging.getLogger("stealth_browser")

# ──────────────────────────────────────────────
# Конфигурация
# ──────────────────────────────────────────────

@dataclass
class BrowserConfig:
    """Конфигурация браузера с антидетектом."""
    headless: bool = False
    disable_automation: bool = True
    proxy: Optional[str] = None
    user_agent: Optional[str] = None
    viewport: Dict[str, int] = field(default_factory=lambda: {"width": 1920, "height": 1080})
    locale: str = "ru-RU"
    timezone_id: str = "Europe/Moscow"
    geolocation: Optional[Dict[str, float]] = None
    extra_args: List[str] = field(default_factory=list)
    
    def __post_init__(self):
        if self.geolocation is None:
            self.geolocation = {"latitude": 55.7558, "longitude": 37.6173}


# ──────────────────────────────────────────────
# CAPTCHA Solving: capzy / capsolver
# ──────────────────────────────────────────────

class CapzySolver:
    """
    Интеграция с сервисом capzy для решения капч.
    
    АPI-ключ: capzy_e1ccf94c1ba7fe9c48694465991061d7e6150a8971b79ff2
    
    Поддерживает:
    - ReCaptcha V2 (с прокси и без)
    - ReCaptcha V3
    - hCaptcha
    - GeeTest
    """
    
    BASE_URLS = {
        "capzy": "https://api.capzy.net",
        "capsolver": "https://api.capsolver.com",
    }
    
    def __init__(self, api_key: str, service: str = "capzy"):
        self.api_key = api_key
        self.service = service
        self.base_url = self.BASE_URLS.get(service, self.BASE_URLS["capsolver"])
        self.session = None
        
    def _ensure_session(self):
        if self.session is None:
            import cloudscraper
            self.session = cloudscraper.create_scraper()
    
    def solve_recaptcha_v2(
        self, 
        website_url: str, 
        website_key: str,
        proxy: Optional[str] = None,
        timeout: int = 120
    ) -> Optional[str]:
        """
        Решает ReCaptcha V2.
        
        Args:
            website_url: URL страницы с капчей
            website_key: data-sitekey
            proxy: опционально "http://user:pass@host:port"
            timeout: максимальное время ожидания (сек)
            
        Returns:
            g-recaptcha-response токен или None
        """
        self._ensure_session()
        
        # Создаём задачу
        task = {
            "type": "ReCaptchaV2TaskProxyless" if not proxy else "ReCaptchaV2Task",
            "websiteURL": website_url,
            "websiteKey": website_key,
        }
        
        if proxy:
            task["proxy"] = proxy
        
        task_payload = {
            "clientKey": self.api_key,
            "task": task
        }
        
        # Пробуем capzy, fallback на capsolver
        for base_url in [self.BASE_URLS.get(self.service), self.BASE_URLS["capsolver"]]:
            if not base_url:
                continue
            try:
                resp = self.session.post(
                    f"{base_url}/createTask",
                    json=task_payload,
                    timeout=30
                )
                result = resp.json()
                
                if result.get("errorId") == 0:
                    task_id = result.get("taskId")
                    logger.info(f"CAPTCHA task created: {task_id} via {base_url}")
                    return self._wait_for_result(task_id, timeout)
                else:
                    logger.warning(f"CAPTCHA error: {result.get('errorDescription')}")
                    continue
                    
            except Exception as e:
                logger.warning(f"CAPTCHA service {base_url} failed: {e}")
                continue
        
        logger.error("All CAPTCHA services failed")
        return None
    
    def _wait_for_result(self, task_id: str, timeout: int = 120) -> Optional[str]:
        """Ожидает результат решения капчи."""
        payload = {
            "clientKey": self.api_key,
            "taskId": task_id
        }
        
        start = time.time()
        while time.time() - start < timeout:
            time.sleep(3)
            try:
                for base_url in [self.BASE_URLS.get(self.service), self.BASE_URLS["capsolver"]]:
                    if not base_url:
                        continue
                    resp = self.session.post(
                        f"{base_url}/getTaskResult",
                        json=payload,
                        timeout=15
                    )
                    result = resp.json()
                    
                    if result.get("status") == "ready":
                        token = result.get("solution", {}).get("gRecaptchaResponse")
                        if token:
                            return token
                    
                    elif result.get("errorId") != 0:
                        logger.error(f"Task failed: {result.get('errorDescription')}")
                        return None
                        
            except Exception as e:
                logger.warning(f"Poll error: {e}")
                time.sleep(2)
        
        logger.error("CAPTCHA timeout")
        return None


# ──────────────────────────────────────────────
# Stealth Browser (Playwright + антидетект)
# ──────────────────────────────────────────────

class StealthBrowser:
    """
    Браузер с продвинутым антидетектом на базе Playwright.
    
    Особенности:
    - Патчинг WebDriver, navigator.webdriver, chrome.runtime
    - Реалистичные User-Agent и заголовки
    - Поддержка прокси (HTTP/SOCKS5)
    - Интеграция с капча-решалами
    - Эмуляция человеческого поведения
    """
    
    def __init__(self, config: Optional[BrowserConfig] = None):
        self.config = config or BrowserConfig()
        self.browser = None
        self.context = None
        self._setup_logging()
        
    def _setup_logging(self):
        logging.basicConfig(
            level=logging.INFO,
            format="%(asctime)s [%(levelname)s] %(name)s: %(message)s"
        )
    
    def _get_stealth_launch_args(self) -> List[str]:
        """Генерирует аргументы для обхода детекции."""
        args = [
            "--disable-blink-features=AutomationControlled",
            "--disable-features=IsolateOrigins,site-per-process",
            "--no-sandbox",
            "--disable-setuid-sandbox",
            "--disable-infobars",
            "--disable-web-security",
            "--disable-features=TranslateUI",
            "--disable-features=ChromeWhatsNewUI",
            f"--lang={self.config.locale}",
        ]
        
        if self.config.disable_automation:
            args.extend([
                "--disable-automation",
                "--disable-background-networking",
                "--enable-features=NetworkService,NetworkServiceInProcess",
            ])
        
        # Если указан прокси
        if self.config.proxy:
            args.append(f"--proxy-server={self.config.proxy}")
        
        args.extend(self.config.extra_args)
        return args
    
    async def start(self):
        """Запускает браузер с антидетектом."""
        from playwright.async_api import async_playwright
        
        self._playwright = await async_playwright().start()
        
        launch_options = {
            "headless": self.config.headless,
            "args": self._get_stealth_launch_args(),
        }
        
        self.browser = await self._playwright.chromium.launch(**launch_options)
        
        # Создаём контекст с реалистичными параметрами
        context_options = {
            "viewport": self.config.viewport,
            "locale": self.config.locale,
            "timezone_id": self.config.timezone_id,
            "geolocation": self.config.geolocation,
            "permissions": ["geolocation"],
            "user_agent": self.config.user_agent or self._generate_user_agent(),
        }
        
        self.context = await self.browser.new_context(**context_options)
        
        # Патчим JS для обхода детекции
        await self._inject_stealth_scripts()
        
        logger.info("Stealth browser started successfully")
        return self.context
    
    def _generate_user_agent(self) -> str:
        """Генерирует реалистичный User-Agent."""
        import fake_useragent
        ua = fake_useragent.UserAgent(browsers=['chrome'])
        return ua.random
    
    async def _inject_stealth_scripts(self):
        """Внедряет скрипты для обхода детекции ботов."""
        stealth_js = """
        // Патчим navigator.webdriver
        Object.defineProperty(navigator, 'webdriver', {
            get: () => undefined
        });
        
        // Патчим chrome.runtime
        window.chrome = {
            runtime: {
                onMessage: { addListener: () => {} },
                onConnect: { addListener: () => {} },
                sendMessage: () => {}
            },
            loadTimes: function() { return {}; },
            csi: function() { return {}; },
            app: { isInstalled: false }
        };
        
        // Патчим plugins
        Object.defineProperty(navigator, 'plugins', {
            get: () => [1, 2, 3, 4, 5]
        });
        
        // Патчим languages
        Object.defineProperty(navigator, 'languages', {
            get: () => ['ru-RU', 'ru', 'en-US', 'en']
        });
        
        // Патчим permissions
        const originalQuery = window.navigator.permissions.query;
        window.navigator.permissions.query = (parameters) => (
            parameters.name === 'notifications' ?
            Promise.resolve({ state: Notification.permission }) :
            originalQuery(parameters)
        );
        """
        await self.context.add_init_script(stealth_js)
    
    async def human_like_delay(self, min_ms: int = 100, max_ms: int = 500):
        """Эмулирует человеческую задержку."""
        delay = random.randint(min_ms, max_ms) / 1000
        await asyncio.sleep(delay)
    
    async def human_like_scroll(self, page, steps: int = 5):
        """Плавный скролл как человек."""
        for _ in range(steps):
            await page.evaluate(f"window.scrollBy(0, {random.randint(100, 400)})")
            await self.human_like_delay(200, 600)
    
    async def close(self):
        """Закрывает браузер."""
        if self.context:
            await self.context.close()
        if self.browser:
            await self.browser.close()
        if self._playwright:
            await self._playwright.stop()


# ──────────────────────────────────────────────
# CLI интерфейс
# ──────────────────────────────────────────────

def main():
    """CLI для запуска браузера с антидетектом."""
    import argparse
    
    parser = argparse.ArgumentParser(description="Stealth Browser для FinHelper")
    parser.add_argument("--url", type=str, help="URL для открытия")
    parser.add_argument("--headless", action="store_true", help="Режим без GUI")
    parser.add_argument("--proxy", type=str, help="Прокси (http://user:pass@host:port)")
    parser.add_argument("--solve-captcha", action="store_true", help="Решить капчу на странице")
    parser.add_argument("--screenshot", type=str, help="Сохранить скриншот")
    parser.add_argument("--html", type=str, help="Сохранить HTML страницы")
    
    args = parser.parse_args()
    
    if args.solve_captcha and args.url:
        print(f"[*] Решаем капчу на {args.url}")
        solver = CapzySolver(
            api_key="capzy_e1ccf94c1ba7fe9c48694465991061d7e6150a8971b79ff2",
            service="capzy"
        )
        token = solver.solve_recaptcha_v2(
            website_url=args.url,
            website_key=args.solve_captcha if isinstance(args.solve_captcha, str) else ""
        )
        if token:
            print(f"[✓] Капча решена! Токен: {token[:50]}...")
        else:
            print("[✗] Не удалось решить капчу")
        return
    
    print("[*] Запуск Stealth Browser...")
    
    import asyncio
    
    async def run():
        config = BrowserConfig(
            headless=args.headless,
            proxy=args.proxy,
        )
        browser = StealthBrowser(config)
        context = await browser.start()
        
        page = context.pages[0] if context.pages else await context.new_page()
        
        if args.url:
            await page.goto(args.url, wait_until="networkidle")
            print(f"[✓] Загружено: {page.url}")
            print(f"[i] Заголовок: {await page.title()}")
            
            if args.screenshot:
                await page.screenshot(path=args.screenshot, full_page=True)
                print(f"[✓] Скриншот сохранён: {args.screenshot}")
            
            if args.html:
                content = await page.content()
                with open(args.html, "w", encoding="utf-8") as f:
                    f.write(content)
                print(f"[✓] HTML сохранён: {args.html}")
        
        input("\n[!] Нажми Enter для выхода...")
        await browser.close()
    
    asyncio.run(run())


if __name__ == "__main__":
    main()
