from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parents[1]
DIST = ROOT / "apps" / "web" / "dist"
NGINX_CONFIG = ROOT / "apps" / "web" / "nginx.conf"
FORBIDDEN_MARKERS = {
    b"demo-admin-pass": "demo login credential",
    b"http://mock.local": "fixture API origin",
    b"Mock endpoint not implemented": "fixture API implementation",
    "QUTCraft CMS 项目正式启动".encode(): "fixture content",
    b"QUTCraft Minecraft Mock": "simulated server status",
    "开发 Mock".encode(): "development model label",
}
FORBIDDEN_PATTERNS = {
    re.compile(rb"https?://(?:localhost|127\.0\.0\.1)(?::\d+)?", re.IGNORECASE): "loopback service origin",
}


def main() -> int:
    if not DIST.is_dir():
        print("PRODUCTION_WEB_GUARD_FAILED: apps/web/dist does not exist", file=sys.stderr)
        return 1

    violations: list[str] = []
    nginx_payload = NGINX_CONFIG.read_bytes()
    image_policies = re.findall(rb"img-src\s+([^;]+);", nginx_payload, re.IGNORECASE)
    if not image_policies or any(b"https:" not in policy for policy in image_policies):
        violations.append("apps/web/nginx.conf does not allow HTTPS images in every Content-Security-Policy")

    for path in DIST.rglob("*"):
        if not path.is_file():
            continue
        payload = path.read_bytes()
        for marker, label in FORBIDDEN_MARKERS.items():
            if marker in payload:
                violations.append(f"{path.relative_to(ROOT)} contains {label}")
        for pattern, label in FORBIDDEN_PATTERNS.items():
            if pattern.search(payload):
                violations.append(f"{path.relative_to(ROOT)} contains {label}")

    if violations:
        print("PRODUCTION_WEB_GUARD_FAILED:", file=sys.stderr)
        for violation in violations:
            print(f"- {violation}", file=sys.stderr)
        return 1

    print("PRODUCTION_WEB_GUARD_OK: production bundle contains no fixture data, development labels, or loopback service origins")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
