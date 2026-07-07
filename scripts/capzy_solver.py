#!/usr/bin/env python3
"""
capzy_solver.py — REST-клиент для Capzy CAPTCHA сервиса.

Поддерживает:
- Создание задач (createTask) для решения reCAPTCHA v2 (без прокси)
- Получение результата (getTaskResult)
- Режим CLI: python capzy_solver.py --solve --sitekey KEY --url URL
- Автоматический fallback на capsolver.com при недоступности capzy.net

API ключ передаётся через аргумент --key, переменную окружения CAPZY_API_KEY
или берётся из конфига по умолчанию.
"""

import argparse
import json
import os
import sys
import time
import urllib.error
import urllib.request
from urllib.parse import urlparse

# ── Defaults ──────────────────────────────────────────────────────────────
DEFAULT_CLIENT_KEY = "capzy_e1ccf94c1ba7fe9c48694465991061d7e6150a8971b79ff2"
BASE_URL_CAPZY = "https://api.capzy.net"
BASE_URL_CAPSOLVER = "https://api.capsolver.com"

POLL_INTERVAL = 3.0        # сек между попытками получить результат
MAX_POLL_SECONDS = 120.0   # общий таймаут


# ── Исключения ───────────────────────────────────────────────────────────
class CapzyError(Exception):
    """Базовое исключение для ошибок CAPTCHA-сервиса."""
    pass


class CapzyAPIError(CapzyError):
    """Ошибка, возвращённая самим API (errorId != 0 или errorCode)."""
    pass


class CapzyTimeoutError(CapzyError):
    """Таймаут ожидания решения капчи."""
    pass


class CapzyNetworkError(CapzyError):
    """Сетевая ошибка при обращении к сервису."""
    pass


# ── Клиент ────────────────────────────────────────────────────────────────
class CapzyClient:
    """
    Клиент для CAPTCHA-сервисов (Capzy / Capsolver).

    Параметры:
        client_key (str): API-ключ.
        base_url (str): Базовый URL сервиса (по умолчанию capzy.net).
        auto_fallback (bool): При ошибках сети автоматически переключаться
                              на capsolver.com (похожий формат API).
    """

    def __init__(
        self,
        client_key: str = DEFAULT_CLIENT_KEY,
        base_url: str = BASE_URL_CAPZY,
        auto_fallback: bool = True,
    ):
        self.client_key = client_key
        self.base_url = base_url.rstrip("/")
        self.auto_fallback = auto_fallback
        self._fallback_used = False

    # ── Низкоуровневый HTTP ───────────────────────────────────────────────

    def _api_call(self, endpoint: str, payload: dict, timeout: int = 30) -> dict:
        """
        Отправляет POST-запрос к API сервиса и возвращает dict-ответ.
        Поднимает CapzyNetworkError при сетевых проблемах.
        """
        url = f"{self.base_url}/{endpoint.lstrip('/')}"
        data = json.dumps(payload).encode("utf-8")

        req = urllib.request.Request(
            url,
            data=data,
            headers={
                "Content-Type": "application/json",
                "Accept": "application/json",
            },
            method="POST",
        )

        try:
            with urllib.request.urlopen(req, timeout=timeout) as resp:
                body = resp.read().decode("utf-8")
                return json.loads(body)
        except urllib.error.HTTPError as e:
            body = e.read().decode("utf-8", errors="replace")
            raise CapzyNetworkError(
                f"HTTP {e.code} от {url}: {body[:500]}"
            ) from e
        except (urllib.error.URLError, OSError, TimeoutError) as e:
            raise CapzyNetworkError(
                f"Сетевая ошибка при обращении к {url}: {e}"
            ) from e
        except json.JSONDecodeError as e:
            raise CapzyNetworkError(
                f"Невалидный JSON в ответе от {url}: {e}"
            ) from e

    def _check_error(self, response: dict, endpoint: str):
        """Проверяет ответ API на наличие ошибок."""
        # Capsolver / Capzy используют errorId (0 = успех)
        error_id = response.get("errorId", None)
        if error_id is not None and error_id != 0:
            error_code = response.get("errorCode", "UNKNOWN")
            error_desc = response.get("errorDescription", "")
            raise CapzyAPIError(
                f"[{error_code}] {error_desc} (endpoint: {endpoint})"
            )

        # Некоторые эндпоинты возвращают error поле
        if "error" in response and response["error"]:
            raise CapzyAPIError(
                f"API error: {response['error']} (endpoint: {endpoint})"
            )

    # ── Основные методы ───────────────────────────────────────────────────

    def create_task(
        self,
        website_url: str,
        website_key: str,
        task_type: str = "RecaptchaV2TaskProxyless",
        **extra,
    ) -> dict:
        """
        Создаёт задачу на решение капчи.

        Аргументы:
            website_url: URL страницы с капчей.
            website_key: Site-key reCAPTCHA (data-sitekey).
            task_type: Тип задачи (по умолч. RecaptchaV2TaskProxyless).
            **extra: Дополнительные поля задачи (isInvisible, pageAction и т.д.)

        Возвращает:
            dict с полями taskId / task_id.

        При сетевой ошибке и auto_fallback=True — переключается на
        capsolver.com и повторяет запрос.
        """
        payload = {
            "clientKey": self.client_key,
            "task": {
                "type": task_type,
                "websiteURL": website_url,
                "websiteKey": website_key,
            },
        }

        # Добавляем любые дополнительные поля в task
        if extra:
            payload["task"].update(extra)

        try:
            response = self._api_call("createTask", payload)
        except CapzyNetworkError as e:
            if self.auto_fallback and not self._fallback_used:
                return self._fallback_to_capsolver("createTask", payload)
            raise

        self._check_error(response, "createTask")
        return response

    def get_task_result(self, task_id: str) -> dict:
        """
        Получает результат решения задачи.

        Возвращает dict с полем solution (например solution['gRecaptchaResponse']).
        Если задача ещё не решена — возвращает {'status': 'processing'}.
        """
        payload = {
            "clientKey": self.client_key,
            "taskId": task_id,
        }

        try:
            response = self._api_call("getTaskResult", payload)
        except CapzyNetworkError as e:
            if self.auto_fallback and not self._fallback_used:
                return self._fallback_to_capsolver("getTaskResult", payload)
            raise

        self._check_error(response, "getTaskResult")
        return response

    def solve_captcha(self, website_url: str, website_key: str, **extra) -> str:
        """
        Полный цикл: создаёт задачу и ждёт её решения.

        Возвращает gRecaptchaResponse (токен) в виде строки.
        При превышении лимита времени — CapzyTimeoutError.
        """
        # 1. Создаём задачу
        create_resp = self.create_task(website_url, website_key, **extra)

        # Capzy возвращает taskId в camelCase
        task_id = create_resp.get("taskId")
        if not task_id:
            # Некоторые сервисы возвращают task_id через model
            task_data = create_resp.get("task", {})
            task_id = task_data.get("id") or create_resp.get("id")
        if not task_id:
            raise CapzyAPIError(
                f"Не удалось получить taskId из ответа: {create_resp}"
            )

        # 2. Ожидаем решения
        deadline = time.monotonic() + MAX_POLL_SECONDS

        while time.monotonic() < deadline:
            result = self.get_task_result(task_id)
            status = result.get("status", "")

            if status == "ready":
                solution = result.get("solution", {})
                token = (
                    solution.get("gRecaptchaResponse")
                    or solution.get("token")
                    or solution.get("captchaToken")
                    or solution.get("text")
                    or solution.get("answer")
                )
                if token:
                    return token
                # Если токена нет, но статус ready — возвращаем solution целиком
                return json.dumps(solution)

            elif status in ("processing", None, ""):
                time.sleep(POLL_INTERVAL)
                continue

            else:
                raise CapzyAPIError(
                    f"Неизвестный статус задачи {task_id}: {status}"
                )

        raise CapzyTimeoutError(
            f"Таймаут {MAX_POLL_SECONDS}с при ожидании решения задачи {task_id}"
        )

    # ── Fallback ───────────────────────────────────────────────────────────

    def _fallback_to_capsolver(self, endpoint: str, payload: dict) -> dict:
        """
        Переключает клиент на capsolver.com и повторяет запрос.
        """
        old_url = self.base_url
        self.base_url = BASE_URL_CAPSOLVER
        self._fallback_used = True
        print(
            f"[!] {old_url} недоступен, переключаемся на {self.base_url}",
            file=sys.stderr,
        )
        return self._api_call(endpoint, payload)


# ── CLI ────────────────────────────────────────────────────────────────────
def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Capzy CAPTCHA Solver — клиент для решения капч",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=(
            "Примеры:\n"
            "  python capzy_solver.py --solve --sitekey 6Le-wxAaAAAAAEHk1FiVnE1L-HEB6nFaA6pBnF8v \\\n"
            "      --url https://example.com\n"
            "  python capzy_solver.py --solve --sitekey KEY --url URL --task-type RecaptchaV3TaskProxyless\n"
            "  python capzy_solver.py --task-id abc-123 --poll\n"
        ),
    )
    parser.add_argument(
        "--key",
        default=os.environ.get("CAPZY_API_KEY") or DEFAULT_CLIENT_KEY,
        help="API-ключ (по умолч. из CAPZY_API_KEY или встроенный)",
    )
    parser.add_argument(
        "--base-url",
        default=BASE_URL_CAPZY,
        help=f"Базовый URL API (по умолч. {BASE_URL_CAPZY})",
    )
    parser.add_argument(
        "--no-fallback",
        action="store_true",
        help="Отключить авто-fallback на capsolver.com",
    )

    # Режимы
    parser.add_argument(
        "--solve",
        action="store_true",
        help="Режим: решить капчу (требует --sitekey и --url)",
    )
    parser.add_argument(
        "--task-id",
        help="ID задачи для ручного опроса результата",
    )
    parser.add_argument(
        "--poll",
        action="store_true",
        help="Опрашивать результат задачи (с --task-id)",
    )

    # Параметры задачи
    parser.add_argument("--sitekey", help="Site-key reCAPTCHA")
    parser.add_argument("--url", help="URL страницы с капчей")
    parser.add_argument(
        "--task-type",
        default="RecaptchaV2TaskProxyless",
        help="Тип задачи (по умолч. RecaptchaV2TaskProxyless)",
    )
    parser.add_argument(
        "--invisible",
        action="store_true",
        help="Invisible reCAPTCHA (для v2)",
    )
    parser.add_argument(
        "--action",
        help="pageAction (для reCAPTCHA v3)",
    )
    parser.add_argument(
        "--page-action",
        dest="action",
        help="pageAction (для reCAPTCHA v3)",
    )

    return parser


def main():
    parser = build_parser()
    args = parser.parse_args()

    # ── Создаём клиент ────────────────────────────────────────────────────
    client = CapzyClient(
        client_key=args.key,
        base_url=args.base_url,
        auto_fallback=not args.no_fallback,
    )

    # ── Режим: решение капчи ──────────────────────────────────────────────
    if args.solve:
        if not args.sitekey or not args.url:
            parser.error("--solve требует --sitekey и --url")

        extra = {}
        if args.invisible:
            extra["isInvisible"] = True
        if args.action:
            extra["pageAction"] = args.action

        try:
            print(f"[*] Создаём задачу (тип: {args.task_type})...", file=sys.stderr)
            token = client.solve_captcha(
                website_url=args.url,
                website_key=args.sitekey,
                task_type=args.task_type,
                **extra,
            )
            print(token)  # Только токен в stdout для pipe/скриптов
        except CapzyError as e:
            print(f"[✗] {e}", file=sys.stderr)
            sys.exit(1)

    # ── Режим: ручной опрос результата ────────────────────────────────────
    elif args.task_id and args.poll:
        try:
            result = client.get_task_result(args.task_id)
            print(json.dumps(result, indent=2, ensure_ascii=False))
        except CapzyError as e:
            print(f"[✗] {e}", file=sys.stderr)
            sys.exit(1)

    # ── Режим: создать задачу (без ожидания) ──────────────────────────────
    elif args.sitekey and args.url:
        extra = {}
        if args.invisible:
            extra["isInvisible"] = True
        if args.action:
            extra["pageAction"] = args.action

        try:
            result = client.create_task(
                website_url=args.url,
                website_key=args.sitekey,
                task_type=args.task_type,
                **extra,
            )
            print(json.dumps(result, indent=2, ensure_ascii=False))
        except CapzyError as e:
            print(f"[✗] {e}", file=sys.stderr)
            sys.exit(1)

    else:
        parser.print_help()


if __name__ == "__main__":
    main()
