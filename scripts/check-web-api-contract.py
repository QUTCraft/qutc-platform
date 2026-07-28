#!/usr/bin/env python3
"""Check that every real frontend API client request exists in OpenAPI."""

from __future__ import annotations

import re
import sys
from pathlib import Path

try:
    import yaml
except ImportError as exc:  # pragma: no cover - dependency guidance
    raise SystemExit("缺少 PyYAML，请先执行：python -m pip install pyyaml") from exc


ROOT = Path(__file__).resolve().parents[1]
OPENAPI = ROOT / "docs" / "api" / "openapi.yaml"
API_DIR = ROOT / "apps" / "web" / "src" / "api"
CLIENT_FILES = ("auth.ts", "invitations.ts", "portal.ts", "admin.ts")
METHODS = {
    "get": "GET",
    "getPage": "GET",
    "post": "POST",
    "patch": "PATCH",
    "del": "DELETE",
    "upload": "POST",
}
CALL_PATTERN = re.compile(
    r"\b(?P<helper>getPage|get|post|patch|del|upload)"
    r"(?:<.*>)?\(\s*"
    r"(?:withQuery\(\s*)?"
    r"(?P<path>`[^`]*`|'[^']*'|\"[^\"]*\"|portalBase)"
)


def canonical(path: str) -> str:
    path = path.split("?", 1)[0]
    path = re.sub(r"\{[A-Za-z0-9_]+\}", "{}", path)
    return path.rstrip("/") or "/"


def documented_routes(spec: dict) -> set[tuple[str, str]]:
    routes: set[tuple[str, str]] = set()
    for path, path_item in spec.get("paths", {}).items():
        for method in METHODS.values():
            operation = path_item.get(method.lower())
            if operation is not None:
                routes.add((method, canonical(path)))
    return routes


def normalize_client_path(expression: str) -> str:
    if expression == "portalBase":
        return canonical("/api/v1/portal/organizations/{organization_slug}")

    value = expression[1:-1]
    value = value.replace(
        "${portalBase}",
        "/api/v1/portal/organizations/{organization_slug}",
    )
    value = value.replace("${adminBase}", "/api/v1/admin")
    value = re.sub(r"\$\{suffix.*$", "", value)
    value = re.sub(r"\$\{encodeURIComponent\([^)]+\)\}", "{parameter}", value)
    value = re.sub(r"\$\{[^}]+\}", "{parameter}", value)
    return canonical(value)


def main() -> int:
    spec = yaml.safe_load(OPENAPI.read_text(encoding="utf-8"))
    documented = documented_routes(spec)
    requests: list[tuple[str, str, str, int]] = []
    unparsed_calls: list[str] = []

    for filename in CLIENT_FILES:
        path = API_DIR / filename
        source = path.read_text(encoding="utf-8")
        for line_number, line in enumerate(source.splitlines(), start=1):
            helper_mentions = re.findall(
                r"\b(?:getPage|get|post|patch|del|upload)(?:<.*>)?\(",
                line,
            )
            matches = list(CALL_PATTERN.finditer(line))
            if helper_mentions and not matches:
                unparsed_calls.append(f"{filename}:{line_number}: {line.strip()}")
            for match in matches:
                requests.append(
                    (
                        METHODS[match.group("helper")],
                        normalize_client_path(match.group("path")),
                        filename,
                        line_number,
                    )
                )

    missing = [
        request
        for request in requests
        if (request[0], request[1]) not in documented
    ]
    if unparsed_calls or missing:
        print("WEB_API_CONTRACT_FAILED")
        if unparsed_calls:
            print("无法解析的前端 API 调用：")
            for call in unparsed_calls:
                print(f"- {call}")
        if missing:
            print("前端已调用但 OpenAPI 未声明：")
            for method, path, filename, line_number in missing:
                print(f"- {method} {path} ({filename}:{line_number})")
        return 1

    unique_requests = {(method, path) for method, path, _, _ in requests}
    print(
        "WEB_API_CONTRACT_OK: "
        f"{len(unique_requests)} frontend requests match OpenAPI "
        f"across {len(CLIENT_FILES)} clients"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
