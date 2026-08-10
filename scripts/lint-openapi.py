#!/usr/bin/env python3
"""Apply repository-specific structural and security lint rules to OpenAPI."""

from __future__ import annotations

import re
import sys
from pathlib import Path
from typing import Any

try:
    import yaml
except ImportError as exc:  # pragma: no cover - dependency guidance
    raise SystemExit("缺少 PyYAML，请先执行：python -m pip install pyyaml") from exc


ROOT = Path(__file__).resolve().parents[1]
OPENAPI = ROOT / "docs" / "api" / "openapi.yaml"
HTTP_METHODS = {"get", "post", "put", "patch", "delete", "head", "options"}
OPERATION_ID_PATTERN = re.compile(r"^[a-z][A-Za-z0-9]+$")


class UniqueKeyLoader(yaml.SafeLoader):
    """Reject duplicate YAML keys instead of silently accepting drift."""


def construct_unique_mapping(
    loader: UniqueKeyLoader,
    node: yaml.nodes.MappingNode,
    deep: bool = False,
) -> dict:
    mapping = {}
    for key_node, value_node in node.value:
        key = loader.construct_object(key_node, deep=deep)
        if key in mapping:
            raise ValueError(f"OpenAPI YAML 存在重复键：{key}")
        mapping[key] = loader.construct_object(value_node, deep=deep)
    return mapping


UniqueKeyLoader.add_constructor(
    yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG,
    construct_unique_mapping,
)


def resolve_local_ref(spec: dict, reference: str) -> Any:
    if not reference.startswith("#/"):
        raise ValueError(f"仅允许仓库内 OpenAPI 引用：{reference}")

    current: Any = spec
    for raw_part in reference[2:].split("/"):
        part = raw_part.replace("~1", "/").replace("~0", "~")
        if not isinstance(current, dict) or part not in current:
            raise ValueError(f"OpenAPI 引用不存在：{reference}")
        current = current[part]
    return current


def walk_references(spec: dict, value: Any) -> None:
    if isinstance(value, dict):
        reference = value.get("$ref")
        if reference is not None:
            if not isinstance(reference, str):
                raise ValueError("$ref 必须是字符串")
            resolve_local_ref(spec, reference)
        for nested in value.values():
            walk_references(spec, nested)
    elif isinstance(value, list):
        for nested in value:
            walk_references(spec, nested)


def lint_schema(name: str, schema: dict) -> list[str]:
    errors: list[str] = []
    required = schema.get("required", [])
    properties = schema.get("properties", {})

    if required and not isinstance(required, list):
        errors.append(f"Schema {name} 的 required 必须是数组")
    if isinstance(required, list) and len(required) != len(set(required)):
        errors.append(f"Schema {name} 的 required 存在重复字段")
    if isinstance(required, list) and isinstance(properties, dict):
        missing = sorted(set(required) - set(properties))
        if missing:
            errors.append(f"Schema {name} 的 required 未声明属性：{', '.join(missing)}")

    enum = schema.get("enum")
    if isinstance(enum, list) and len(enum) != len(set(map(str, enum))):
        errors.append(f"Schema {name} 的 enum 存在重复值")
    if schema.get("type") == "array" and "items" not in schema:
        errors.append(f"Schema {name} 是数组但缺少 items")
    return errors


def main() -> int:
    try:
        spec = yaml.load(OPENAPI.read_text(encoding="utf-8"), Loader=UniqueKeyLoader)
        if not isinstance(spec, dict):
            raise ValueError("OpenAPI 根节点必须是对象")

        errors: list[str] = []
        version = str(spec.get("openapi", ""))
        if not version.startswith("3.1."):
            errors.append(f"openapi 必须使用 3.1.x，当前为 {version or '未声明'}")

        info = spec.get("info", {})
        for field in ("title", "version", "description"):
            if not info.get(field):
                errors.append(f"info.{field} 不能为空")

        declared_tags = {
            tag.get("name")
            for tag in spec.get("tags", [])
            if isinstance(tag, dict) and tag.get("name")
        }
        if not declared_tags:
            errors.append("必须声明顶层 tags")

        security_schemes = spec.get("components", {}).get("securitySchemes", {})
        bearer = security_schemes.get("BearerAuth", {})
        if bearer.get("type") != "http" or bearer.get("scheme") != "bearer":
            errors.append("components.securitySchemes.BearerAuth 必须是 HTTP Bearer")
        access_cookie = security_schemes.get("AccessCookie", {})
        if (
            access_cookie.get("type") != "apiKey"
            or access_cookie.get("in") != "cookie"
            or access_cookie.get("name") != "qutc_access"
        ):
            errors.append("components.securitySchemes.AccessCookie 必须声明 qutc_access Cookie")

        operation_ids: set[str] = set()
        operation_count = 0
        for path, path_item in spec.get("paths", {}).items():
            if not isinstance(path_item, dict):
                errors.append(f"路径项必须是对象：{path}")
                continue
            for method, operation in path_item.items():
                if method.lower() not in HTTP_METHODS:
                    continue
                operation_count += 1
                label = f"{method.upper()} {path}"
                if not isinstance(operation, dict):
                    errors.append(f"{label} 操作必须是对象")
                    continue

                operation_id = operation.get("operationId")
                if not operation_id:
                    errors.append(f"{label} 缺少 operationId")
                elif not OPERATION_ID_PATTERN.fullmatch(str(operation_id)):
                    errors.append(f"{label} 的 operationId 不是 lowerCamelCase：{operation_id}")
                elif operation_id in operation_ids:
                    errors.append(f"{label} 的 operationId 重复：{operation_id}")
                else:
                    operation_ids.add(operation_id)

                if not operation.get("summary"):
                    errors.append(f"{label} 缺少 summary")
                tags = operation.get("tags", [])
                if not tags:
                    errors.append(f"{label} 缺少 tags")
                for tag in tags:
                    if tag not in declared_tags:
                        errors.append(f"{label} 使用了未声明 tag：{tag}")

                responses = operation.get("responses", {})
                if not isinstance(responses, dict) or not responses:
                    errors.append(f"{label} 缺少 responses")
                elif not any(str(status).startswith("2") for status in responses):
                    errors.append(f"{label} 缺少成功的 2xx response")

                security = operation.get("security", spec.get("security"))
                if path.startswith("/api/v1/admin/"):
                    expected_security = [{"BearerAuth": []}, {"AccessCookie": []}]
                    if security != expected_security:
                        errors.append(f"{label} 必须显式允许 BearerAuth 或 AccessCookie")
                    for status in ("401", "403"):
                        if status not in responses:
                            errors.append(f"{label} 缺少 {status} 安全响应")
                elif path.startswith("/api/v1/portal/") and security not in (None, []):
                    errors.append(f"{label} 属于公开 Portal API，不得要求后台认证")

        schemas = spec.get("components", {}).get("schemas", {})
        for name, schema in schemas.items():
            if isinstance(schema, dict):
                errors.extend(lint_schema(name, schema))

        walk_references(spec, spec)
        if errors:
            print("OPENAPI_LINT_FAILED")
            for error in errors:
                print(f"- {error}")
            return 1

        print(
            "OPENAPI_LINT_OK: "
            f"{operation_count} operations, {len(schemas)} schemas, "
            f"{len(declared_tags)} tags"
        )
        return 0
    except (OSError, TypeError, ValueError, yaml.YAMLError) as exc:
        print(f"OPENAPI_LINT_FAILED: {exc}")
        return 1


if __name__ == "__main__":
    sys.exit(main())
