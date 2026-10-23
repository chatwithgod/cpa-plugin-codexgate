"""Package an existing versioned Windows DLL without overwriting releases."""
from pathlib import Path
import argparse
import hashlib
import json
import re
import zipfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", default="0.1.0-dev.3")
    args = parser.parse_args()
    if not re.fullmatch(r"\d+\.\d+\.\d+(?:-[A-Za-z0-9.]+)?", args.version):
        raise SystemExit("Invalid version")
    root = Path(__file__).resolve().parent.parent
    dll = root / "build" / args.version / f"codexgate-v{args.version}.dll"
    if not dll.is_file():
        raise SystemExit("Build the DLL first")
    out = root / "dist" / f"codexgate-v{args.version}-windows-amd64.zip"
    out.parent.mkdir(exist_ok=True)
    digest = hashlib.sha256(dll.read_bytes()).hexdigest()
    manifest = {"plugin_id": "codexgate", "version": args.version, "platform": "windows-amd64", "tested_cpa": "8.0.38", "abi": 1, "schema": 6, "dll": dll.name, "sha256": digest, "stage": "development; production Codex acceptance pending"}
    files = [dll, root / "README.md", root / "LICENSE", root / "THIRD_PARTY_NOTICES.md", root / "docs" / "VERIFICATION.md", *sorted((root / "licenses").iterdir())]
    # Exclusive creation preserves every previously generated archive.
    with zipfile.ZipFile(out, "x", zipfile.ZIP_DEFLATED) as archive:
        for path in files:
            archive.write(path, dll.name if path == dll else path.relative_to(root).as_posix())
        archive.writestr("build-info.json", json.dumps(manifest, indent=2))
        archive.writestr("SHA256SUMS", f"{digest}  {dll.name}\n")
    with zipfile.ZipFile(out) as archive:
        assert hashlib.sha256(archive.read(dll.name)).hexdigest() == digest
        assert archive.testzip() is None
    print(json.dumps({"archive": str(out), "size": out.stat().st_size, "sha256": hashlib.sha256(out.read_bytes()).hexdigest()}, ensure_ascii=False))


if __name__ == "__main__":
    main()
