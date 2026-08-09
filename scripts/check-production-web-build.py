from pathlib import Path
import sys


ROOT = Path(__file__).resolve().parents[1]
DIST = ROOT / "apps" / "web" / "dist"
FORBIDDEN_MARKERS = {
    b"demo-admin-pass": "demo login credential",
    b"http://mock.local": "fixture API origin",
    b"Mock endpoint not implemented": "fixture API implementation",
    "QUTCraft CMS 项目正式启动".encode(): "fixture content",
    b"QUTCraft Minecraft Mock": "simulated server status",
    "开发 Mock".encode(): "development model label",
}


def main() -> int:
    if not DIST.is_dir():
        print("PRODUCTION_WEB_GUARD_FAILED: apps/web/dist does not exist", file=sys.stderr)
        return 1

    violations: list[str] = []
    for path in DIST.rglob("*"):
        if not path.is_file():
            continue
        payload = path.read_bytes()
        for marker, label in FORBIDDEN_MARKERS.items():
            if marker in payload:
                violations.append(f"{path.relative_to(ROOT)} contains {label}")

    if violations:
        print("PRODUCTION_WEB_GUARD_FAILED:", file=sys.stderr)
        for violation in violations:
            print(f"- {violation}", file=sys.stderr)
        return 1

    print("PRODUCTION_WEB_GUARD_OK: production bundle contains no fixture data or development model labels")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
