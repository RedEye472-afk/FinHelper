#!/usr/bin/env python3
"""
FinHelper Anti-Detect Module
=============================
Утилиты для обхода детекции ботов и анти-ИИ средств.

Компоненты:
- Генерация реалистичных фингерпринтов браузера
- Ротация User-Agent'ов и заголовков
- Эмуляция человеческого поведения (клики, скролл, тайпинг)
- Обход Cloudflare, Akamai, Imperva
- Маскировка трафика
"""

import os
import re
import json
import time
import random
import logging
import hashlib
from typing import Optional, Dict, List, Tuple
from datetime import datetime

logger = logging.getLogger("anti_detect")


# ──────────────────────────────────────────────
# User-Agent и заголовки
# ──────────────────────────────────────────────

class UserAgentManager:
    """
    Менеджер User-Agent'ов с ротацией.
    Генерирует реалистичные UA для разных браузеров и платформ.
    """
    
    CHROME_VERSIONS = list(range(120, 135))
    FIREFOX_VERSIONS = [f"{x}.0" for x in range(115, 135)]
    EDGE_VERSIONS = list(range(120, 135))
    
    # Реальные User-Agent строки
    REAL_AGENTS = [
        # Chrome 131+ Windows
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36",
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
        # Edge
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/132.0.0.0",
        # Firefox
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0",
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:134.0) Gecko/20100101 Firefox/134.0",
    ]
    
    def __init__(self):
        self._current = None
        self._blacklist = set()
    
    def get_random(self, browser: str = "chrome") -> str:
        """Возвращает случайный User-Agent."""
        import fake_useragent
        try:
            ua = fake_useragent.UserAgent(browsers=[browser], os='windows')
            return ua.random
        except:
            return random.choice(self.REAL_AGENTS)
    
    def rotate(self) -> str:
        """Ротирует User-Agent."""
        self._current = self.get_random()
        return self._current
    
    def get_headers(self, referer: Optional[str] = None) -> Dict[str, str]:
        """Генерирует реалистичные заголовки HTTP."""
        ua = self._current or self.rotate()
        
        headers = {
            "User-Agent": ua,
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
            "Accept-Language": "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7",
            "Accept-Encoding": "gzip, deflate, br",
            "Sec-Ch-Ua": self._get_sec_ch_ua(ua),
            "Sec-Ch-Ua-Mobile": "?0",
            "Sec-Ch-Ua-Platform": "\"Windows\"",
            "Sec-Fetch-Dest": "document",
            "Sec-Fetch-Mode": "navigate",
            "Sec-Fetch-Site": "none",
            "Sec-Fetch-User": "?1",
            "Upgrade-Insecure-Requests": "1",
            "Dnt": "1",
            "Connection": "keep-alive",
        }
        
        if referer:
            headers["Referer"] = referer
            headers["Sec-Fetch-Site"] = "same-origin"
        
        return headers
    
    def _get_sec_ch_ua(self, ua: str) -> str:
        """Извлекает Sec-CH-UA из User-Agent."""
        if "Edg/" in ua:
            return '"Microsoft Edge";v="131", "Chromium";v="131", "Not_A Brand";v="24"'
        elif "Firefox" in ua:
            return ""
        else:
            return '"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"'


# ──────────────────────────────────────────────
# Прокси и ротация IP
# ──────────────────────────────────────────────

class ProxyManager:
    """Менеджер прокси с ротацией."""
    
    def __init__(self, proxies: Optional[List[str]] = None):
        self.proxies = proxies or []
        self._current_index = -1
    
    def add_proxy(self, proxy: str):
        """Добавляет прокси в пул."""
        self.proxies.append(proxy)
    
    def get_next(self) -> Optional[str]:
        """Возвращает следующий прокси из пула (round-robin)."""
        if not self.proxies:
            return None
        self._current_index = (self._current_index + 1) % len(self.proxies)
        return self.proxies[self._current_index]
    
    def get_random(self) -> Optional[str]:
        """Возвращает случайный прокси."""
        if not self.proxies:
            return None
        return random.choice(self.proxies)


# ──────────────────────────────────────────────
# Эмуляция человеческого поведения
# ──────────────────────────────────────────────

class HumanEmulator:
    """
    Эмуляция человеческого поведения в браузере.
    
    Задержки, движения мыши, скролл, тайпинг — 
    всё с реалистичными случайными интервалами.
    """
    
    def __init__(self):
        self._rng = random.Random()
    
    def random_delay(self, min_ms: float = 50, max_ms: float = 300) -> float:
        """Генерирует случайную задержку в миллисекундах."""
        return self._rng.uniform(min_ms, max_ms) / 1000
    
    def typing_delay(self, text_length: int) -> float:
        """
        Генерирует задержку для печати текста.
        Симулирует скорость печати человека (200-400 символов/мин).
        """
        chars_per_sec = self._rng.uniform(3.0, 6.0)
        return text_length / chars_per_sec
    
    def scroll_pattern(self, page_height: int) -> List[Tuple[int, float]]:
        """
        Генерирует паттерн скролла как у человека.
        Возвращает список (позиция_scroll, задержка_в_сек).
        """
        pattern = []
        current_pos = 0
        
        while current_pos < page_height:
            # Человек скроллит неравномерно
            step = self._rng.randint(100, 500)
            current_pos = min(current_pos + step, page_height)
            delay = self._rng.uniform(0.3, 1.5)
            pattern.append((current_pos, delay))
            
            # Иногда останавливается почитать
            if self._rng.random() < 0.3:
                pattern.append((current_pos, self._rng.uniform(2.0, 5.0)))
        
        return pattern


# ──────────────────────────────────────────────
# Обход Cloudflare и WAF
# ──────────────────────────────────────────────

class CloudflareBypass:
    """
    Инструменты для обхода Cloudflare, Akamai, Imperva.
    """
    
    @staticmethod
    def get_cloudscraper():
        """
        Возвращает scraper, который обходит Cloudflare.
        """
        import cloudscraper
        return cloudscraper.create_scraper(
            browser={
                'browser': 'chrome',
                'platform': 'windows',
                'desktop': True,
            },
            delay=15  # задержка для обхода JS challenge
        )
    
    @staticmethod
    def create_session_with_stealth():
        """
        Создаёт httpx сессию с антидетект заголовками.
        """
        import httpx
        
        ua_mgr = UserAgentManager()
        headers = ua_mgr.get_headers()
        
        transport = httpx.HTTPTransport(
            retries=3,
        )
        
        return httpx.Client(
            headers=headers,
            transport=transport,
            timeout=30.0,
            follow_redirects=True,
            http2=True,
        )


# ──────────────────────────────────────────────
# Маскировка и анти-ИИ
# ──────────────────────────────────────────────

class ContentMasker:
    """
    Инструменты для маскировки контента от ИИ-детекторов.
    """
    
    @staticmethod
    def add_human_noise(text: str, intensity: float = 0.02) -> str:
        """
        Добавляет "человеческий шум" в текст.
        - Типографические ошибки (очень редко)
        - Разнообразие длины предложений
        - Естественные паузы
        """
        if random.random() > intensity:
            return text
        
        # Замена некоторых знаков на похожие
        replacements = {
            'a': 'а',  # латинская a на кириллицу
            'e': 'е',  # латинская e на кириллицу
            'o': 'о',  # латинская o на кириллицу
            'c': 'с',  # латинская c на кириллицу
        }
        
        result = list(text)
        for i, char in enumerate(result):
            if char in replacements and random.random() < 0.01:
                result[i] = replacements[char]
        
        return ''.join(result)
    
    @staticmethod
    def vary_punctuation(text: str) -> str:
        """Варьирует пунктуацию для натуральности."""
        # Замена ... на — и наоборот в редких случаях
        if random.random() < 0.1:
            text = text.replace('...', '—')
            text = text.replace('...', '…')
        return text


# ──────────────────────────────────────────────
# CLI интерфейс
# ──────────────────────────────────────────────

def main():
    """CLI для тестирования антидетект инструментов."""
    import argparse
    
    parser = argparse.ArgumentParser(description="Anti-Detect Toolkit для FinHelper")
    parser.add_argument("--test-ua", action="store_true", help="Тест генерации User-Agent")
    parser.add_argument("--test-headers", action="store_true", help="Тест генерации заголовков")
    parser.add_argument("--test-bypass", metavar="URL", help="Тест обхода Cloudflare для URL")
    
    args = parser.parse_args()
    
    if args.test_ua:
        ua_mgr = UserAgentManager()
        print("=== Тест User-Agent ===")
        for browser in ["chrome", "firefox", "edge"]:
            for _ in range(2):
                print(f"[{browser}] {ua_mgr.get_random(browser)}")
    
    if args.test_headers:
        ua_mgr = UserAgentManager()
        headers = ua_mgr.get_headers()
        print("=== Тест заголовков ===")
        for k, v in headers.items():
            print(f"  {k}: {v}")
    
    if args.test_bypass:
        print(f"=== Тест обхода {args.test_bypass} ===")
        bypass = CloudflareBypass()
        scraper = bypass.get_cloudscraper()
        try:
            resp = scraper.get(args.test_bypass, timeout=30)
            print(f"[✓] Статус: {resp.status_code}")
            print(f"[✓] Размер: {len(resp.content)} bytes")
        except Exception as e:
            print(f"[✗] Ошибка: {e}")


if __name__ == "__main__":
    main()
