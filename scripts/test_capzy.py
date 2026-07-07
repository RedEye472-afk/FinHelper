#!/usr/bin/env python3
"""
test_capzy.py — демо-тест для capzy_solver.py

Запуск:
    python test_capzy.py                    # базовые юнит-тесты (без сети)
    python test_capzy.py --live             # живой тест с отправкой запроса
    python test_capzy.py --help             # полные опции

Живой тест отправляет задачу на capzy.net (или capsolver.com при fallback)
с публичным тестовым sitekey и проверяет, что получен токен.
"""

import argparse
import json
import sys
import time
import unittest
from unittest.mock import patch, MagicMock

# Добавляем путь к модулю
sys.path.insert(0, r"C:\Users\user\ZCodeProject\FinHelper\scripts")

from capzy_solver import (
    CapzyClient,
    CapzyError,
    CapzyAPIError,
    CapzyTimeoutError,
    CapzyNetworkError,
    BASE_URL_CAPZY,
    BASE_URL_CAPSOLVER,
    DEFAULT_CLIENT_KEY,
    MAX_POLL_SECONDS,
    POLL_INTERVAL,
)


# ═══════════════════════════════════════════════════════════════════════════
# Юнит-тесты
# ═══════════════════════════════════════════════════════════════════════════
class TestCapzyClientUnit(unittest.TestCase):
    """Тесты без реальных HTTP-вызовов (mocks)."""

    def setUp(self):
        self.client = CapzyClient(
            client_key="test_key_123",
            base_url=BASE_URL_CAPZY,
            auto_fallback=False,
        )

    # ── create_task ───────────────────────────────────────────────────────

    @patch("capzy_solver.CapzyClient._api_call")
    def test_create_task_success(self, mock_api_call):
        mock_api_call.return_value = {
            "errorId": 0,
            "taskId": "test-task-456",
        }
        result = self.client.create_task(
            website_url="https://example.com",
            website_key="6Le-wxAaAAAAAEHk1FiVnE1L-HEB6nFaA6pBnF8v",
        )

        self.assertEqual(result["taskId"], "test-task-456")
        # Проверяем, что _api_call вызван с правильными аргументами
        mock_api_call.assert_called_once()
        endpoint, payload = mock_api_call.call_args[0]
        self.assertEqual(endpoint, "createTask")
        self.assertEqual(payload["clientKey"], "test_key_123")
        self.assertEqual(
            payload["task"]["type"], "RecaptchaV2TaskProxyless"
        )

    @patch("capzy_solver.CapzyClient._api_call")
    def test_create_task_with_extra(self, mock_api_call):
        mock_api_call.return_value = {"errorId": 0, "taskId": "t-789"}
        self.client.create_task(
            website_url="https://example.com",
            website_key="abc123",
            task_type="RecaptchaV3TaskProxyless",
            pageAction="verify",
        )
        _, payload = mock_api_call.call_args[0]
        self.assertEqual(payload["task"]["type"], "RecaptchaV3TaskProxyless")
        self.assertEqual(payload["task"]["pageAction"], "verify")

    @patch("capzy_solver.CapzyClient._api_call")
    def test_create_task_api_error(self, mock_api_call):
        mock_api_call.return_value = {
            "errorId": 1,
            "errorCode": "ERROR_INVALID_TASK_DATA",
            "errorDescription": "Invalid site key",
        }
        with self.assertRaises(CapzyAPIError) as ctx:
            self.client.create_task(
                website_url="https://example.com",
                website_key="bad-key",
            )
        self.assertIn("ERROR_INVALID_TASK_DATA", str(ctx.exception))

    @patch("capzy_solver.CapzyClient._api_call")
    def test_create_task_network_error(self, mock_api_call):
        mock_api_call.side_effect = CapzyNetworkError("Connection refused")
        with self.assertRaises(CapzyNetworkError):
            self.client.create_task(
                website_url="https://example.com",
                website_key="abc",
            )

    # ── get_task_result ───────────────────────────────────────────────────

    @patch("capzy_solver.CapzyClient._api_call")
    def test_get_task_result_processing(self, mock_api_call):
        mock_api_call.return_value = {
            "errorId": 0,
            "status": "processing",
        }
        result = self.client.get_task_result("task-111")
        self.assertEqual(result["status"], "processing")

    @patch("capzy_solver.CapzyClient._api_call")
    def test_get_task_result_ready(self, mock_api_call):
        mock_api_call.return_value = {
            "errorId": 0,
            "status": "ready",
            "solution": {
                "gRecaptchaResponse": "03AGdBq25...token...",
            },
        }
        result = self.client.get_task_result("task-222")
        sol = result["solution"]
        self.assertEqual(sol["gRecaptchaResponse"], "03AGdBq25...token...")

    # ── solve_captcha (полный цикл) ──────────────────────────────────────

    @patch("capzy_solver.CapzyClient._api_call")
    def test_solve_captcha_success(self, mock_api_call):
        # Первый вызов: createTask → возвращает taskId
        # Второй вызов: getTaskResult → processing
        # Третий вызов: getTaskResult → ready с токеном
        mock_api_call.side_effect = [
            {"errorId": 0, "taskId": "task-solve-1"},
            {"errorId": 0, "status": "processing"},
            {
                "errorId": 0,
                "status": "ready",
                "solution": {"gRecaptchaResponse": "TOKEN_OK"},
            },
        ]

        token = self.client.solve_captcha(
            website_url="https://example.com",
            website_key="test-key",
        )
        self.assertEqual(token, "TOKEN_OK")
        self.assertEqual(mock_api_call.call_count, 3)

    @patch("capzy_solver.CapzyClient._api_call")
    def test_solve_captcha_timeout(self, mock_api_call):
        """Если сервер всё время отвечает processing — должен быть таймаут."""
        def side_effect(endpoint, payload, **kw):
            if "createTask" in endpoint:
                return {"errorId": 0, "taskId": "task-timeout"}
            return {"errorId": 0, "status": "processing"}

        mock_api_call.side_effect = side_effect

        # Патчим MAX_POLL_SECONDS для быстрого теста
        with patch("capzy_solver.MAX_POLL_SECONDS", 0.1):
            with patch("capzy_solver.POLL_INTERVAL", 0.05):
                with self.assertRaises(CapzyTimeoutError):
                    self.client.solve_captcha(
                        website_url="https://example.com",
                        website_key="test-key",
                    )

    # ── Fallback ──────────────────────────────────────────────────────────

    @patch("capzy_solver.CapzyClient._api_call")
    def test_auto_fallback_on_network_error(self, mock_api_call):
        """При сетевой ошибке и auto_fallback=True клиент переключается
        на capsolver.com и повторяет запрос."""
        client = CapzyClient(
            client_key="fallback_key",
            base_url=BASE_URL_CAPZY,
            auto_fallback=True,
        )

        # Первый вызов (capzy) — сетевой сбой, второй (capsolver) — успех
        mock_api_call.side_effect = [
            CapzyNetworkError("timeout"),
            {"errorId": 0, "taskId": "capsolver-task"},
        ]

        result = client.create_task(
            website_url="https://example.com",
            website_key="key",
        )
        self.assertEqual(result["taskId"], "capsolver-task")
        self.assertTrue(client._fallback_used)
        # _api_call был вызван дважды
        self.assertEqual(mock_api_call.call_count, 2)

    # ── CLI аргументы ─────────────────────────────────────────────────────

    def test_cli_help(self):
        """Проверяем, что парсер CLI создаётся без ошибок."""
        from capzy_solver import build_parser

        parser = build_parser()
        # Проверяем, что все аргументы на месте
        actions = {a.dest for a in parser._actions}
        for expected in ("solve", "sitekey", "url", "task_type", "key",
                         "base_url", "no_fallback", "task_id", "poll"):
            self.assertIn(expected, actions, f"Missing CLI arg: {expected}")


# ═══════════════════════════════════════════════════════════════════════════
# Интеграционный тест (live)
# ═══════════════════════════════════════════════════════════════════════════
def run_live_test(base_url: str = BASE_URL_CAPZY):
    """
    Отправляет реальную задачу на API и проверяет ответ.
    Используется публичный тестовый sitekey.
    """
    client = CapzyClient(
        client_key=DEFAULT_CLIENT_KEY,
        base_url=base_url,
        auto_fallback=True,
    )

    # Публичный тестовый ключ от Google
    test_sitekey = "6Le-wxAaAAAAAEHk1FiVnE1L-HEB6nFaA6pBnF8v"
    test_url = "https://www.google.com/recaptcha/api2/demo"

    print(f"[*] Базовый URL: {client.base_url}")
    print(f"[*] SiteKey: {test_sitekey}")
    print(f"[*] URL:     {test_url}")
    print()

    # Шаг 1: создаём задачу
    print("[1] Создаём задачу...")
    try:
        create_resp = client.create_task(
            website_url=test_url,
            website_key=test_sitekey,
        )
        task_id = create_resp.get("taskId")
        print(f"    ✓ taskId = {task_id}")
        if client._fallback_used:
            print("    (используется capsolver.com)")
        print(f"    Полный ответ: {json.dumps(create_resp, indent=4, ensure_ascii=False)}")
    except CapzyError as e:
        print(f"    ✗ Ошибка: {e}")
        return False

    print()

    # Шаг 2: опрашиваем результат (макс. 5 попыток)
    print(f"[2] Ожидаем решения (макс. 5 попыток)...")
    for attempt in range(1, 6):
        time.sleep(3)
        try:
            result = client.get_task_result(task_id)
            status = result.get("status", "unknown")
            print(f"    Попытка {attempt}: status = {status}")

            if status == "ready":
                solution = result.get("solution", {})
                token = (
                    solution.get("gRecaptchaResponse")
                    or solution.get("token")
                    or solution.get("text")
                )
                if token:
                    preview = token[:50] + "..." if len(token) > 50 else token
                    print(f"    ✓ Токен получен: {preview}")
                else:
                    print(f"    ✓ Статус ready, решение: {json.dumps(solution, indent=2)}")
                print()
                return True

        except CapzyError as e:
            print(f"    ✗ Ошибка: {e}")
            break

    print("    ✗ Таймаут или ошибка")
    return False


# ═══════════════════════════════════════════════════════════════════════════
# Точка входа
# ═══════════════════════════════════════════════════════════════════════════
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Тест capzy_solver.py")
    parser.add_argument(
        "--live",
        action="store_true",
        help="Выполнить живой тест с реальным API",
    )
    parser.add_argument(
        "--base-url",
        default=BASE_URL_CAPZY,
        help=f"Базовый URL API (по умолч. {BASE_URL_CAPZY})",
    )
    args, remaining = parser.parse_known_args()

    if args.live:
        success = run_live_test(base_url=args.base_url)
        sys.exit(0 if success else 1)
    else:
        # Юнит-тесты
        unittest.main(argv=sys.argv[:1] + remaining, verbosity=2)
