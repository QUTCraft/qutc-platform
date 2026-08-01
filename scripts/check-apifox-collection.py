#!/usr/bin/env python3
"""Validate the importable Apifox/Postman smoke collection against OpenAPI."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path
from typing import Any, Iterator

try:
    import yaml
except ImportError as exc:  # pragma: no cover - dependency guidance
    raise SystemExit("缺少 PyYAML，请先执行：python -m pip install pyyaml") from exc


ROOT = Path(__file__).resolve().parents[1]
OPENAPI = ROOT / "docs" / "api" / "openapi.yaml"
COLLECTION = ROOT / "docs" / "api" / "apifox" / "core-smoke.postman_collection.json"
ENVIRONMENT = ROOT / "docs" / "api" / "apifox" / "local.postman_environment.json"
HTTP_METHODS = {"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
SENSITIVE_MARKERS = ("password", "token", "secret", "key")


def canonical(path: str) -> str:
    path = re.sub(r"^https?://[^/]+", "", path)
    path = path.replace("{{baseUrl}}", "")
    path = path.split("?", 1)[0]
    path = re.sub(r"\{\{[^}]+\}\}", "{}", path)
    path = re.sub(r"\{[A-Za-z0-9_]+\}", "{}", path)
    return path.rstrip("/") or "/"


def collection_requests(items: list[dict]) -> Iterator[tuple[str, str, str]]:
    for item in items:
        if "item" in item:
            yield from collection_requests(item["item"])
            continue
        request = item.get("request", {})
        method = str(request.get("method", "")).upper()
        raw_url = request.get("url", {}).get("raw", "")
        yield item.get("name", "<unnamed>"), method, canonical(raw_url)


def documented_routes(spec: dict) -> set[tuple[str, str]]:
    return {
        (method.upper(), canonical(path))
        for path, path_item in spec.get("paths", {}).items()
        for method in path_item
        if method.upper() in HTTP_METHODS
    }


def main() -> int:
    try:
        spec = yaml.safe_load(OPENAPI.read_text(encoding="utf-8"))
        collection: dict[str, Any] = json.loads(COLLECTION.read_text(encoding="utf-8"))
        environment: dict[str, Any] = json.loads(ENVIRONMENT.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError, yaml.YAMLError) as exc:
        print(f"APIFOX_COLLECTION_FAILED: {exc}")
        return 1

    documented = documented_routes(spec)
    requests = list(collection_requests(collection.get("item", [])))
    errors: list[str] = []
    for name, method, path in requests:
        if method not in HTTP_METHODS:
            errors.append(f"{name} 使用非法 HTTP 方法：{method or '<empty>'}")
        elif (method, path) not in documented:
            errors.append(f"{name} 未匹配 OpenAPI：{method} {path}")

    for variable in environment.get("values", []):
        key = str(variable.get("key", "")).lower()
        value = str(variable.get("value", ""))
        if any(marker in key for marker in SENSITIVE_MARKERS) and value:
            errors.append(f"环境模板中的敏感变量必须留空：{variable.get('key')}")

    required_variables = {
        "baseUrl",
        "organizationSlug",
        "adminEmail",
        "adminPassword",
        "accessToken",
        "runId",
        "contentId",
        "applicationId",
        "activityPlanId",
    }
    present_variables = {
        variable.get("key")
        for variable in environment.get("values", [])
        if variable.get("enabled", True)
    }
    missing_variables = sorted(required_variables - present_variables)
    if missing_variables:
        errors.append(f"环境模板缺少变量：{', '.join(missing_variables)}")

    if errors:
        print("APIFOX_COLLECTION_FAILED")
        for error in errors:
            print(f"- {error}")
        return 1

    print(
        "APIFOX_COLLECTION_OK: "
        f"{len(requests)} requests match OpenAPI; sensitive values are empty"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
