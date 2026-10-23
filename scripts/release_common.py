"""Shared paths and metadata for the five native release targets (stdlib only)."""
from pathlib import Path
import hashlib
import json
import platform
import re

ROOT = Path(__file__).resolve().parent.parent
PINS = json.loads((ROOT / "scripts/toolchains.json").read_text(encoding="utf-8"))
TARGETS = ("windows-amd64", "linux-amd64", "linux-arm64", "darwin-amd64", "darwin-arm64")
LIBRARIES = {"windows": "codexgate.dll", "linux": "codexgate.so", "darwin": "codexgate.dylib"}


def version(value=None):
    value = value or (ROOT / "VERSION").read_text(encoding="utf-8").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", value):
        raise ValueError("Official release version must be three dotted numbers")
    return value


def native_target():
    system = platform.system().lower()
    machine = platform.machine().lower()
    arch = {"x86_64": "amd64", "amd64": "amd64", "arm64": "arm64", "aarch64": "arm64"}.get(machine)
    target = f"{system}-{arch}"
    if target not in TARGETS:
        raise ValueError(f"Unsupported native target: {target}")
    return system, arch


def sha256(path):
    with Path(path).open("rb") as f:
        return hashlib.file_digest(f, "sha256").hexdigest()


def write_new_json(path, value):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("x", encoding="utf-8", newline="\n") as f:
        json.dump(value, f, ensure_ascii=False, indent=2)
        f.write("\n")
