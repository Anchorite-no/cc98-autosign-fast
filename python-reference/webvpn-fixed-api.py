#!/usr/bin/env python3
# -*- coding: utf-8 -*-

from __future__ import annotations

import argparse
import json
import re
import sys
import time
from pathlib import Path
from typing import Any

MISSING_DEPENDENCIES: list[str] = []

try:
    import requests
except ModuleNotFoundError:
    MISSING_DEPENDENCIES.append("requests")
    requests = None  # type: ignore[assignment]

try:
    from Crypto.Cipher import AES
except ModuleNotFoundError:
    MISSING_DEPENDENCIES.append("pycryptodome")
    AES = None  # type: ignore[assignment]


LOGIN_URL = "https://webvpn.zju.edu.cn/login"
DO_LOGIN_URL = "https://webvpn.zju.edu.cn/do-login"
DO_CONFIRM_LOGIN_URL = "https://webvpn.zju.edu.cn/do-confirm-login"
WEBVPN_BASE = "https://webvpn.zju.edu.cn"
DEFAULT_ENV_PATH = ".env"
DEFAULT_COOKIE_CACHE_PATH = ".webvpn-cookie-cache.json"
DEFAULT_CONNECT_TIMEOUT_SEC = 5.0
DEFAULT_REQUEST_TIMEOUT_SEC = 15.0
DEFAULT_ROUTE_VALUE = "8768cab8c7e7ee1c6799ad807f94da0a"

DEFAULT_TOKEN_PATH = (
    "/https/77726476706e69737468656265737421ffe744922e3426537d51d1e2974724/"
    "connect/token?vpn-12-o2-www.cc98.org"
)
DEFAULT_SIGN_PATH = (
    "/https/77726476706e69737468656265737421f1e748d22433310830079bab/"
    "me/signin?vpn-12-o2-www.cc98.org"
)

CSRF_PATTERN = re.compile(r'name="_csrf"\s+value="([^"]+)"')
CAPTCHA_PATTERN = re.compile(r'name="captcha_id"\s+value="([^"]+)"')
KEY_IV_PATTERN = re.compile(r'encrypt\s*\([^,]+,\s*"([^"]+)"\s*,\s*"([^"]+)"\s*\)')


def configure_stdio() -> None:
    for stream in (sys.stdout, sys.stderr):
        if hasattr(stream, "reconfigure"):
            try:
                stream.reconfigure(encoding="utf-8")
            except (ValueError, OSError):
                pass


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Sign in to CC98 through fixed WebVPN API routes using .env credentials."
    )
    parser.add_argument("--env", default=DEFAULT_ENV_PATH, help="Path to .env file. Default: .env")
    parser.add_argument("--webvpn-user", help="WebVPN username override.")
    parser.add_argument("--webvpn-pass", help="WebVPN password override.")
    parser.add_argument("--cc98-user", help="Run with a single CC98 username override.")
    parser.add_argument("--cc98-pass", help="Run with a single CC98 password override.")
    parser.add_argument(
        "--account-index",
        type=int,
        help="Run only one indexed account from .env (1-based).",
    )
    parser.add_argument(
        "--token-path",
        help="Fixed WebVPN token API path. Defaults to .env or built-in value.",
    )
    parser.add_argument(
        "--sign-path",
        help="Fixed WebVPN sign API path. Defaults to .env or built-in value.",
    )
    return parser.parse_args()


def ensure_dependencies() -> None:
    if not MISSING_DEPENDENCIES:
        return
    missing = ", ".join(MISSING_DEPENDENCIES)
    raise RuntimeError(
        f"Missing Python dependencies: {missing}. "
        "Run `python3 -m pip install -r requirements.txt` in this directory first."
    )


def parse_env_file(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    if not path.exists():
        return values

    for raw_line in path.read_text(encoding="utf-8").splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        key = key.strip().lstrip("\ufeff")
        value = value.strip()
        if len(value) >= 2 and value[0] == value[-1] and value[0] in {"'", '"'}:
            value = value[1:-1]
        values[key] = value
    return values


def get_first_non_empty(*values: Any) -> str:
    for value in values:
        if value is None:
            continue
        text = str(value).strip()
        if text:
            return text
    return ""


def get_float_setting(env_values: dict[str, str], key: str, default: float) -> float:
    raw = env_values.get(key, "").strip()
    if not raw:
        return default
    try:
        return float(raw)
    except ValueError as exc:
        raise ValueError(f"Invalid float value for {key}: {raw}") from exc


def build_accounts(args: argparse.Namespace, env_values: dict[str, str]) -> list[dict[str, str]]:
    if args.cc98_user or args.cc98_pass:
        if not (args.cc98_user and args.cc98_pass):
            raise ValueError("Both --cc98-user and --cc98-pass are required together")
        return [{"username": args.cc98_user.strip(), "password": args.cc98_pass.strip()}]

    raw_count = env_values.get("CC98_ACCOUNT_COUNT", "").strip()
    if not raw_count:
        raise ValueError("Missing CC98_ACCOUNT_COUNT in .env")

    try:
        count = int(raw_count)
    except ValueError as exc:
        raise ValueError(f"Invalid CC98_ACCOUNT_COUNT: {raw_count}") from exc

    if count < 1:
        raise ValueError("CC98_ACCOUNT_COUNT must be at least 1")

    accounts: list[dict[str, str]] = []
    for index in range(1, count + 1):
        username = env_values.get(f"CC98_USER_{index}", "").strip()
        password = env_values.get(f"CC98_PASS_{index}", "").strip()
        if not username or not password:
            raise ValueError(f"Missing CC98_USER_{index} or CC98_PASS_{index}")
        accounts.append({"username": username, "password": password})

    if args.account_index is not None:
        if args.account_index < 1 or args.account_index > len(accounts):
            raise ValueError("Invalid account index")
        return [accounts[args.account_index - 1]]

    return accounts


def load_settings(args: argparse.Namespace) -> dict[str, Any]:
    env_path = Path(args.env)
    if not env_path.exists():
        raise FileNotFoundError(f".env file not found: {args.env}")

    env_values = parse_env_file(env_path)
    webvpn_user = get_first_non_empty(args.webvpn_user, env_values.get("WEBVPN_USER"))
    webvpn_pass = get_first_non_empty(args.webvpn_pass, env_values.get("WEBVPN_PASS"))
    if not webvpn_user or not webvpn_pass:
        raise ValueError("Missing WEBVPN_USER or WEBVPN_PASS")

    return {
        "webvpn_user": webvpn_user,
        "webvpn_pass": webvpn_pass,
        "accounts": build_accounts(args, env_values),
        "token_path": get_first_non_empty(args.token_path, env_values.get("CC98_TOKEN_PATH"), DEFAULT_TOKEN_PATH),
        "sign_path": get_first_non_empty(args.sign_path, env_values.get("CC98_SIGN_PATH"), DEFAULT_SIGN_PATH),
        "cookie_cache_path": Path(
            get_first_non_empty(env_values.get("WEBVPN_COOKIE_CACHE_FILE"), DEFAULT_COOKIE_CACHE_PATH)
        ),
        "connect_timeout_sec": get_float_setting(
            env_values, "CC98_CONNECT_TIMEOUT_SEC", DEFAULT_CONNECT_TIMEOUT_SEC
        ),
        "request_timeout_sec": get_float_setting(
            env_values, "CC98_REQUEST_TIMEOUT_SEC", DEFAULT_REQUEST_TIMEOUT_SEC
        ),
    }


def extract_fields(html: str) -> tuple[str, str, str, str]:
    csrf_match = CSRF_PATTERN.search(html)
    captcha_match = CAPTCHA_PATTERN.search(html)
    key_iv_match = KEY_IV_PATTERN.search(html)
    if not csrf_match or not captcha_match or not key_iv_match:
        raise ValueError("Failed to extract login page fields")
    return (
        csrf_match.group(1),
        captcha_match.group(1),
        key_iv_match.group(1),
        key_iv_match.group(2),
    )


def encrypt_password(password: str, key: str, iv: str) -> str:
    cipher = AES.new(key.encode("utf-8"), AES.MODE_CFB, iv.encode("utf-8"), segment_size=128)
    encrypted = cipher.encrypt(password.encode("utf-8"))
    return iv.encode("utf-8").hex() + encrypted.hex()


def request_timeout(connect_timeout_sec: float, request_timeout_sec: float) -> tuple[float, float]:
    return (connect_timeout_sec, request_timeout_sec)


def export_cookie_map(session: requests.Session) -> dict[str, str]:
    return {cookie.name: cookie.value for cookie in session.cookies}


def ensure_default_route_cookie(session: requests.Session) -> None:
    if session.cookies.get("route"):
        return
    session.cookies.set("route", DEFAULT_ROUTE_VALUE, domain="webvpn.zju.edu.cn", path="/")


def restore_cookie_map(session: requests.Session, cookies: dict[str, str]) -> None:
    for name, value in cookies.items():
        session.cookies.set(name, value, domain="webvpn.zju.edu.cn", path="/")
    ensure_default_route_cookie(session)


def has_core_webvpn_cookies(session: requests.Session) -> bool:
    cookies = export_cookie_map(session)
    return "wengine_vpn_ticketwebvpn_zju_edu_cn" in cookies


def load_cookie_cache(session: requests.Session, cache_path: Path) -> bool:
    if not cache_path.exists():
        return False

    try:
        cached = json.loads(cache_path.read_text(encoding="utf-8"))
    except (OSError, ValueError, json.JSONDecodeError):
        return False

    cookies = cached.get("cookies")
    if not isinstance(cookies, dict):
        return False

    restore_cookie_map(
        session,
        {str(name): str(value) for name, value in cookies.items() if str(name) and str(value)},
    )
    return has_core_webvpn_cookies(session)


def save_cookie_cache(session: requests.Session, cache_path: Path) -> None:
    payload = {
        "saved_at": int(time.time()),
        "cookies": export_cookie_map(session),
    }
    cache_path.parent.mkdir(parents=True, exist_ok=True)
    cache_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")


def confirm_webvpn(
    session: requests.Session,
    form: dict[str, str],
    connect_timeout_sec: float,
    request_timeout_sec: float,
) -> None:
    resp = session.post(
        DO_CONFIRM_LOGIN_URL,
        data=form,
        timeout=request_timeout(connect_timeout_sec, request_timeout_sec),
    )
    resp.raise_for_status()


def login_webvpn(
    session: requests.Session,
    username: str,
    password: str,
    connect_timeout_sec: float,
    request_timeout_sec: float,
) -> None:
    login_resp = session.get(
        LOGIN_URL,
        timeout=request_timeout(connect_timeout_sec, request_timeout_sec),
    )
    login_resp.raise_for_status()
    csrf, captcha_id, key, iv = extract_fields(login_resp.text)

    form = {
        "_csrf": csrf,
        "auth_type": "local",
        "username": username,
        "sms_code": "",
        "password": encrypt_password(password, key, iv),
        "captcha": "",
        "needCaptcha": "false",
        "captcha_id": captcha_id,
    }

    login_post = session.post(
        DO_LOGIN_URL,
        data=form,
        timeout=request_timeout(connect_timeout_sec, request_timeout_sec),
    )
    login_post.raise_for_status()

    try:
        login_payload = login_post.json()
    except ValueError:
        login_payload = {}

    error = login_payload.get("error")
    if error == "NEED_CONFIRM":
        confirm_webvpn(session, form, connect_timeout_sec, request_timeout_sec)
    elif error:
        message = str(login_payload.get("message") or error)
        raise RuntimeError(f"WebVPN login failed: {message}")


def verify_token(
    session: requests.Session,
    token_path: str,
    cc98_user: str,
    cc98_pass: str,
    connect_timeout_sec: float,
    request_timeout_sec: float,
) -> dict[str, Any]:
    resp = session.post(
        WEBVPN_BASE + token_path,
        data={
            "client_id": "9a1fd200-8687-44b1-4c20-08d50a96e5cd",
            "client_secret": "8b53f727-08e2-4509-8857-e34bf92b27f2",
            "grant_type": "password",
            "username": cc98_user,
            "password": cc98_pass,
        },
        headers={
            "Accept": "application/json, text/plain, */*",
            "Content-Type": "application/x-www-form-urlencoded",
            "Origin": "https://www.cc98.org",
            "Referer": "https://www.cc98.org/",
        },
        timeout=request_timeout(connect_timeout_sec, request_timeout_sec),
    )

    try:
        payload = resp.json()
        if not isinstance(payload, dict):
            payload = {"raw": resp.text[:500]}
    except ValueError:
        payload = {"raw": resp.text[:500]}

    access_token = str(payload.get("access_token", ""))
    return {
        "status": resp.status_code,
        "ok": resp.ok and bool(access_token),
        "access_token": access_token,
        "payload": payload,
        "raw_text": resp.text[:500],
    }


def verify_sign(
    session: requests.Session,
    sign_path: str,
    access_token: str,
    connect_timeout_sec: float,
    request_timeout_sec: float,
) -> dict[str, Any]:
    resp = session.post(
        WEBVPN_BASE + sign_path,
        data="",
        headers={
            "Accept": "*/*",
            "Content-Type": "application/json",
            "Authorization": f"Bearer {access_token}",
            "Origin": "https://www.cc98.org",
            "Referer": "https://www.cc98.org/",
        },
        timeout=request_timeout(connect_timeout_sec, request_timeout_sec),
    )
    body = resp.text.strip()
    payload: dict[str, Any] | None = None
    try:
        json_payload = resp.json()
        if isinstance(json_payload, dict):
            payload = json_payload
    except ValueError:
        payload = None

    return {
        "status": resp.status_code,
        "ok": resp.ok or body == "has_signed_in_today" or body.isdigit(),
        "body": body[:500],
        "payload": payload,
    }


def get_sign_info(
    session: requests.Session,
    sign_path: str,
    access_token: str,
    connect_timeout_sec: float,
    request_timeout_sec: float,
) -> dict[str, Any] | None:
    try:
        resp = session.get(
            WEBVPN_BASE + sign_path,
            headers={
                "Accept": "application/json, text/plain, */*",
                "Authorization": f"Bearer {access_token}",
                "Origin": "https://www.cc98.org",
                "Referer": "https://www.cc98.org/",
            },
            timeout=request_timeout(connect_timeout_sec, request_timeout_sec),
        )
        resp.raise_for_status()
        payload = resp.json()
        return payload if isinstance(payload, dict) else {}
    except requests.RequestException:
        return None


def is_webvpn_login_response(token_result: dict[str, Any]) -> bool:
    raw_text = str(token_result.get("raw_text", "")).strip()
    if not raw_text:
        raw_text = str(token_result.get("payload", {}).get("raw", "")).strip()
    markers = ('name="_csrf"', "captcha_id", "wengine_vpn_ticket", "WebVPN")
    return any(marker in raw_text for marker in markers)


def summarize_sign_result(
    sign_result: dict[str, Any] | None,
    sign_info: dict[str, Any] | None,
) -> dict[str, Any]:
    summary = {
        "status": "failed",
        "reward": None,
        "streak": None,
    }
    if sign_result is None:
        return summary

    body = str(sign_result.get("body", "")).strip()
    payload = sign_result.get("payload") or {}

    if body == "has_signed_in_today":
        summary["status"] = "already"
    elif body.isdigit():
        summary["status"] = "success"
        summary["reward"] = int(body)
    elif isinstance(payload, dict):
        if payload.get("hasSignedInToday") is True:
            summary["status"] = "already" if sign_result["status"] != 200 else "success"
        reward = payload.get("reward") or payload.get("lastReward")
        streak = payload.get("signInCount") or payload.get("lastSignInCount")
        if reward is not None:
            try:
                summary["reward"] = int(reward)
            except (TypeError, ValueError):
                pass
        if streak is not None:
            try:
                summary["streak"] = int(streak)
            except (TypeError, ValueError):
                pass

    if isinstance(sign_info, dict):
        if summary["reward"] is None:
            reward = sign_info.get("lastReward")
            if reward is not None:
                try:
                    summary["reward"] = int(reward)
                except (TypeError, ValueError):
                    pass
        if summary["streak"] is None:
            streak = sign_info.get("lastSignInCount")
            if streak is not None:
                try:
                    summary["streak"] = int(streak)
                except (TypeError, ValueError):
                    pass
        if sign_info.get("hasSignedInToday") is True and summary["status"] == "failed":
            summary["status"] = "already"

    return summary


def format_result_text(summary: dict[str, Any], sign_result: dict[str, Any] | None) -> str:
    if sign_result is None:
        return "❌ 签到失败 · 请求失败"

    status = summary["status"]
    reward = summary["reward"]
    streak = summary["streak"]

    if status in {"already", "success"}:
        parts = ["✅ 签到成功"]
        if reward is not None:
            parts.append(f"🎁 {reward}财富值")
        if streak is not None:
            parts.append(f"📅 连续 {streak} 天")
        return " · ".join(parts)

    body = str(sign_result.get("body", "")).strip()
    reason = body or f"HTTP {sign_result['status']}"
    return f"❌ 签到失败 · {reason}"


def run_account(
    session: requests.Session,
    account: dict[str, str],
    account_index: int,
    token_path: str,
    sign_path: str,
    connect_timeout_sec: float,
    request_timeout_sec: float,
) -> dict[str, Any]:
    token_result = verify_token(
        session,
        token_path,
        account["username"],
        account["password"],
        connect_timeout_sec,
        request_timeout_sec,
    )

    sign_result = None
    sign_info = None
    if sign_path and token_result.get("access_token"):
        sign_result = verify_sign(
            session,
            sign_path,
            token_result["access_token"],
            connect_timeout_sec,
            request_timeout_sec,
        )
        if sign_result["ok"]:
            sign_info = get_sign_info(
                session,
                sign_path,
                token_result["access_token"],
                connect_timeout_sec,
                request_timeout_sec,
            )

    summary = summarize_sign_result(sign_result, sign_info)
    return {
        "index": account_index,
        "username": account["username"],
        "result_text": format_result_text(summary, sign_result),
        "token_result": token_result,
    }


def run_account_with_cookie_retry(
    session: requests.Session,
    account: dict[str, str],
    account_index: int,
    settings: dict[str, Any],
    allow_cookie_retry: bool,
) -> tuple[dict[str, Any], bool]:
    result = run_account(
        session,
        account,
        account_index,
        settings["token_path"],
        settings["sign_path"],
        settings["connect_timeout_sec"],
        settings["request_timeout_sec"],
    )

    if not allow_cookie_retry or not is_webvpn_login_response(result["token_result"]):
        return result, False

    session.cookies.clear()
    login_webvpn(
        session,
        settings["webvpn_user"],
        settings["webvpn_pass"],
        settings["connect_timeout_sec"],
        settings["request_timeout_sec"],
    )
    save_cookie_cache(session, settings["cookie_cache_path"])

    retry_result = run_account(
        session,
        account,
        account_index,
        settings["token_path"],
        settings["sign_path"],
        settings["connect_timeout_sec"],
        settings["request_timeout_sec"],
    )
    return retry_result, True


def format_output_lines(account_results: list[dict[str, Any]]) -> list[str]:
    return [
        f"账号{result['index']} {result['result_text']}"
        for result in account_results
    ]


def main() -> int:
    started = time.perf_counter()
    configure_stdio()
    args = parse_args()
    ensure_dependencies()
    settings = load_settings(args)
    session = requests.Session()

    cache_path = settings["cookie_cache_path"]
    cookie_cache_loaded = load_cookie_cache(session, cache_path)
    cookie_cache_hit = cookie_cache_loaded

    if not cookie_cache_loaded:
        login_webvpn(
            session,
            settings["webvpn_user"],
            settings["webvpn_pass"],
            settings["connect_timeout_sec"],
            settings["request_timeout_sec"],
        )
        save_cookie_cache(session, cache_path)

    account_results: list[dict[str, Any]] = []
    for account_index, account in enumerate(settings["accounts"], start=1):
        result, retried_with_login = run_account_with_cookie_retry(
            session,
            account,
            account_index,
            settings,
            allow_cookie_retry=bool(cookie_cache_loaded and account_index == 1),
        )
        if retried_with_login:
            cookie_cache_hit = False
        account_results.append(result)

    total_seconds = round(time.perf_counter() - started, 2)
    for line in format_output_lines(account_results):
        print(line)
    print(f"Cookie {'✅ 命中' if cookie_cache_hit else '❌ 未命中'}")
    print(f"耗时 ⏱ {total_seconds:.2f}s")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # pragma: no cover
        print(f"[webvpn-fixed-api][ERROR] {exc}", file=sys.stderr)
        raise SystemExit(1)
