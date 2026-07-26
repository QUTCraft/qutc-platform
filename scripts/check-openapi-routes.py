#!/usr/bin/env python3
"""Check that the documented OpenAPI operations match the Gin route table."""

from __future__ import annotations

import re
import sys
from pathlib import Path

try:
    import yaml
except ImportError as exc:  # pragma: no cover - dependency guidance
    raise SystemExit("缺少 PyYAML，请先执行：python -m pip install pyyaml") from exc


ROOT = Path(__file__).resolve().parents[1]
MAIN_GO = ROOT / "apps" / "api" / "cmd" / "server" / "main.go"
OPENAPI = ROOT / "docs" / "api" / "openapi.yaml"
HTTP_METHODS = {"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}


class UniqueKeyLoader(yaml.SafeLoader):
    """Reject duplicate YAML keys instead of silently keeping the last value."""


def construct_unique_mapping(loader: UniqueKeyLoader, node: yaml.nodes.MappingNode, deep: bool = False) -> dict:
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


def canonical(path: str) -> str:
    """Normalize Gin :id and OpenAPI {id} placeholders to one representation."""

    normalized = re.sub(r":([A-Za-z0-9_]+)", "{}", path)
    return re.sub(r"\{[A-Za-z0-9_]+\}", "{}", normalized)


def registered_routes() -> set[tuple[str, str]]:
    source = MAIN_GO.read_text(encoding="utf-8")
    prefixes: dict[str, str] = {"router": ""}

    for match in re.finditer(
        r"(?P<name>[A-Za-z_][A-Za-z0-9_]*)\s*:=\s*"
        r"(?P<parent>[A-Za-z_][A-Za-z0-9_]*)\.Group\(\"(?P<path>[^\"]+)\"",
        source,
    ):
        parent = match.group("parent")
        if parent not in prefixes:
            raise ValueError(f"无法解析路由组前缀：{parent}")
        prefixes[match.group("name")] = prefixes[parent] + match.group("path")

    routes: set[tuple[str, str]] = set()
    for match in re.finditer(
        r"(?P<group>[A-Za-z_][A-Za-z0-9_]*)\.(?P<method>"
        r"GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\(\"(?P<path>[^\"]*)\"",
        source,
    ):
        group = match.group("group")
        if group not in prefixes:
            raise ValueError(f"无法解析路由组前缀：{group}")
        routes.add((match.group("method"), canonical(prefixes[group] + match.group("path"))))
    return routes


def documented_routes(spec: dict) -> set[tuple[str, str]]:
    routes: set[tuple[str, str]] = set()
    operation_ids: set[str] = set()

    def parameter_details(parameter: dict) -> dict:
        reference = parameter.get("$ref")
        if not reference:
            return parameter
        prefix = "#/components/parameters/"
        if not reference.startswith(prefix):
            raise ValueError(f"不支持的参数引用：{reference}")
        name = reference.removeprefix(prefix)
        try:
            return spec["components"]["parameters"][name]
        except KeyError as exc:
            raise ValueError(f"参数引用不存在：{reference}") from exc

    for path, item in spec.get("paths", {}).items():
        for method, operation in item.items():
            method = method.upper()
            if method not in HTTP_METHODS:
                continue
            operation_id = operation.get("operationId")
            if not operation_id:
                raise ValueError(f"缺少 operationId：{method} {path}")
            if operation_id in operation_ids:
                raise ValueError(f"重复 operationId：{operation_id}")
            operation_ids.add(operation_id)
            routes.add((method, canonical(path)))

            path_parameters = {
                parameter_details(parameter)["name"]
                for parameter in operation.get("parameters", [])
                if parameter_details(parameter).get("in") == "path"
            }
            path_item_parameters = {
                parameter_details(parameter)["name"]
                for parameter in item.get("parameters", [])
                if parameter_details(parameter).get("in") == "path"
            }
            declared_parameters = path_parameters | path_item_parameters
            required_parameters = set(re.findall(r"\{([A-Za-z0-9_]+)\}", path))
            missing_parameters = required_parameters - declared_parameters
            if missing_parameters:
                raise ValueError(
                    f"路径参数未声明：{method} {path}: {', '.join(sorted(missing_parameters))}"
                )
    return routes


def main() -> int:
    spec = yaml.load(OPENAPI.read_text(encoding="utf-8"), Loader=UniqueKeyLoader)
    registered = registered_routes()
    documented = documented_routes(spec)
    missing = sorted(registered - documented)
    stale = sorted(documented - registered)

    if missing or stale:
        print(f"路由数：Go={len(registered)}，OpenAPI={len(documented)}")
        if missing:
            print("\nGo 已注册但 OpenAPI 缺失：")
            for method, path in missing:
                print(f"- {method} {path}")
        if stale:
            print("\nOpenAPI 已记录但 Go 未注册：")
            for method, path in stale:
                print(f"- {method} {path}")
        return 1

    print(f"OPENAPI_ROUTES_OK: {len(registered)} routes and operationIds are aligned")
    return 0


if __name__ == "__main__":
    sys.exit(main())
