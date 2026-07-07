#!/usr/bin/env python3
"""
FinHelper Scripts — Comprehensive Test Suite
==============================================
Тестирование всех модулей: антидетект, стелс-браузер, капча.
"""

import os
import sys
import json
import pytest
from unittest.mock import patch, MagicMock

# ──────────────────────────────────────────────
# Anti-Detect Tests
# ──────────────────────────────────────────────

class TestUserAgentManager:
    """Тесты менеджера User-Agent."""
    
    def test_import(self):
        """Проверка импорта модуля."""
        try:
            from anti_detect import UserAgentManager
            assert True
        except ImportError as e:
            pytest.skip(f"Модуль anti_detect не найден: {e}")
    
    def test_get_random(self):
        """Генерация случайного UA."""
        try:
            from anti_detect import UserAgentManager
            uam = UserAgentManager()
            ua = uam.get_random("chrome")
            assert ua is not None
            assert len(ua) > 20
            assert "Mozilla" in ua
        except ImportError:
            pytest.skip("anti_detect module not available")
    
    def test_get_headers(self):
        """Генерация заголовков."""
        try:
            from anti_detect import UserAgentManager
            uam = UserAgentManager()
            headers = uam.get_headers()
            assert "User-Agent" in headers
            assert "Accept" in headers
            assert "Accept-Language" in headers
        except ImportError:
            pytest.skip("anti_detect module not available")


class TestHumanEmulator:
    """Тесты эмуляции человека."""
    
    def test_random_delay(self):
        """Случайная задержка."""
        try:
            from anti_detect import HumanEmulator
            he = HumanEmulator()
            delay = he.random_delay(100, 500)
            assert 0.05 <= delay <= 0.55  # ms to seconds
        except ImportError:
            pytest.skip("anti_detect module not available")
    
    def test_typing_delay(self):
        """Задержка печати."""
        try:
            from anti_detect import HumanEmulator
            he = HumanEmulator()
            delay = he.typing_delay(100)
            assert delay > 0
            assert delay < 60
        except ImportError:
            pytest.skip("anti_detect module not available")
    
    def test_scroll_pattern(self):
        """Паттерн скролла."""
        try:
            from anti_detect import HumanEmulator
            he = HumanEmulator()
            pattern = he.scroll_pattern(3000)
            assert len(pattern) > 0
            for pos, delay in pattern:
                assert 0 <= pos <= 3000
                assert delay > 0
        except ImportError:
            pytest.skip("anti_detect module not available")


class TestContentMasker:
    """Тесты маскировки контента."""
    
    def test_add_human_noise(self):
        """Добавление человеческого шума."""
        try:
            from anti_detect import ContentMasker
            cm = ContentMasker()
            text = "The quick brown fox jumps over the lazy dog"
            masked = cm.add_human_noise(text, intensity=0.5)
            assert len(masked) > 0
        except ImportError:
            pytest.skip("anti_detect module not available")


# ──────────────────────────────────────────────
# Stealth Browser Tests
# ──────────────────────────────────────────────

class TestStealthBrowser:
    """Тесты стелс-браузера."""
    
    def test_import(self):
        """Проверка импорта."""
        try:
            from stealth_browser import StealthBrowser, BrowserConfig, CapzySolver
            assert True
        except ImportError as e:
            pytest.skip(f"stealth_browser module not available: {e}")
    
    def test_browser_config_defaults(self):
        """Конфигурация по умолчанию."""
        try:
            from stealth_browser import BrowserConfig
            config = BrowserConfig()
            assert config.viewport == {"width": 1920, "height": 1080}
            assert config.locale == "ru-RU"
            assert config.timezone_id == "Europe/Moscow"
        except ImportError:
            pytest.skip("stealth_browser module not available")
    
    def test_capzy_solver_init(self):
        """Инициализация CapzySolver."""
        try:
            from stealth_browser import CapzySolver
            solver = CapzySolver(api_key="test_key")
            assert solver.api_key == "test_key"
            assert solver.service == "capzy"
        except ImportError:
            pytest.skip("stealth_browser module not available")


# ──────────────────────────────────────────────
# Integration Tests
# ──────────────────────────────────────────────

class TestCloudflareBypass:
    """Тесты обхода Cloudflare (без реального запроса)."""
    
    def test_get_cloudscraper(self):
        """Создание cloudscraper."""
        try:
            from anti_detect import CloudflareBypass
            from cloudscraper import CloudScraper
            scraper = CloudflareBypass.get_cloudscraper()
            assert isinstance(scraper, CloudScraper)
        except ImportError:
            pytest.skip("cloudscraper not available")


# ──────────────────────────────────────────────
# Capzy API Tests
# ──────────────────────────────────────────────

class TestCapzySolver:
    """Тесты Capzy Solver."""
    
    CAPZY_KEY = "capzy_e1ccf94c1ba7fe9c48694465991061d7e6150a8971b79ff2"
    
    def test_import_capzy(self):
        """Проверка импорта standalone capzy модуля."""
        try:
            # Пробуем разные варианты импорта
            try:
                from capzy_solver import CapzySolver, solve_captcha
            except ImportError:
                from stealth_browser import CapzySolver
                solve_captcha = None
            assert True
        except ImportError:
            pytest.skip("capzy_solver module not available")
    
    @pytest.mark.skip(reason="Требует реального API-ключа и сети")
    def test_real_api_call(self):
        """Реальный вызов API (интеграционный тест)."""
        from stealth_browser import CapzySolver
        solver = CapzySolver(api_key=self.CAPZY_KEY)
        token = solver.solve_recaptcha_v2(
            website_url="https://www.google.com/recaptcha/api2/demo",
            website_key="6LeIxAcTAAAAAJcZVRqyHh71UMIEGNQ_MXjiZKhI",
            timeout=30
        )
        # Этот тест может вернуть None если сервис недоступен
        # или токен если всё ок
        if token:
            print(f"Capcha solved! Token: {token[:50]}...")
        else:
            print("Capcha solve returned None (service may be unavailable)")


# ──────────────────────────────────────────────
# Run Tests
# ──────────────────────────────────────────────

if __name__ == "__main__":
    pytest.main([__file__, "-v", "--tb=short"])
