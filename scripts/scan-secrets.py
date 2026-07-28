#!/usr/bin/env python3
"""Fail on tracked/unignored high-confidence secrets and frontend secret config."""

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
MAX_TEXT_SIZE = 2 << 20
SECRET_PATTERNS = {
    "private key": re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----"),
    "GitHub token": re.compile(r"\bgh[pousr]_[A-Za-z0-9]{30,}\b"),
    "AWS access key": re.compile(r"\b(?:AKIA|ASIA)[A-Z0-9]{16}\b"),
    "Google API key": re.compile(r"\bAIza[0-9A-Za-z_-]{35}\b"),
    "OpenAI-style key": re.compile(r"\bsk-[A-Za-z0-9_-]{24,}\b"),
    "JWT value": re.compile(r"\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b"),
    "credential in URL": re.compile(r"https?://[^/\s:@]+:[^/\s@]+@"),
}
FRONTEND_FORBIDDEN = re.compile(
    r"\b(?:JWT_ACCESS_SECRET|BOOTSTRAP_ADMIN_PASSWORD|MYSQL_PASSWORD|"
    r"REDIS_PASSWORD|MINIO_ROOT_PASSWORD|SMTP_PASSWORD|RCON_PASSWORD)\b"
)


def repository_files() -> list[Path]:
    result = subprocess.run(
        ["git", "ls-files", "--cached", "--others", "--exclude-standard", "-z"],
        cwd=ROOT,
        check=True,
        capture_output=True,
    )
    return [
        ROOT / entry.decode("utf-8")
        for entry in result.stdout.split(b"\0")
        if entry
    ]


def main() -> int:
    findings: list[str] = []
    for path in repository_files():
        try:
            if not path.is_file() or path.stat().st_size > MAX_TEXT_SIZE:
                continue
            raw = path.read_bytes()
            if b"\0" in raw:
                continue
            text = raw.decode("utf-8")
        except (OSError, UnicodeDecodeError):
            continue

        relative = path.relative_to(ROOT).as_posix()
        for line_number, line in enumerate(text.splitlines(), start=1):
            for name, pattern in SECRET_PATTERNS.items():
                if pattern.search(line):
                    findings.append(f"{relative}:{line_number}: detected {name}")
            if relative.startswith("apps/web/") and FRONTEND_FORBIDDEN.search(line):
                findings.append(
                    f"{relative}:{line_number}: server-only secret name appears in frontend"
                )

    if findings:
        print("SECRET_SCAN_FAILED")
        for finding in findings:
            print(f"- {finding}")
        return 1

    print("SECRET_SCAN_OK: no high-confidence secrets or frontend secret config found")
    return 0


if __name__ == "__main__":
    sys.exit(main())
