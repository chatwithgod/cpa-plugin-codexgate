"""Build/package, use the CPA installer, then smoke-test the installed library."""
import argparse
import json
import os
import re
import shutil
import subprocess
import sys
from pathlib import Path
from fetch_tools import fetch_cpa
from release_common import ROOT, LIBRARIES, native_target, version, write_new_json


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--go", default="go")
    parser.add_argument("--cc", default=os.environ.get("CC", "gcc"))
    parser.add_argument("--core")
    parser.add_argument("--skip-build", action="store_true")
    args = parser.parse_args()
    release = version()
    goos, goarch = native_target()
    target = f"{goos}-{goarch}"
    env = os.environ.copy()
    env.update({"CGO_ENABLED": "1", "CC": args.cc, "GOTOOLCHAIN": "local"})
    env.setdefault("GOCACHE", str(ROOT / ".cache/go-build"))
    env.setdefault("GOPATH", str(ROOT / ".cache/gopath"))
    core = Path(args.core).resolve(strict=True) if args.core else fetch_cpa()
    if not args.skip_build:
        subprocess.run([sys.executable, "scripts/build.py", "--go", args.go, "--cc", args.cc], cwd=ROOT, env=env, check=True)
    build = ROOT / "build" / release / target
    library = build / LIBRARIES[goos]
    if goos == "linux":
        output = subprocess.check_output(["readelf", "--version-info", str(library)], text=True)
        versions = sorted(set(re.findall(r"Name: GLIBC_(\d+\.\d+(?:\.\d+)?)", output)), key=lambda s: tuple(map(int, s.split("."))))
        if any(tuple(map(int, v.split("."))) > (2, 17) for v in versions):
            raise SystemExit(f"Library exceeds GLIBC 2.17 baseline: {versions}")
        write_new_json(build / "runtime-compatibility.json", {"glibc_versions": versions, "baseline": "2.17", "passed": True})
    subprocess.run([sys.executable, "scripts/package_release.py"], cwd=ROOT, env=env, check=True)
    out = ROOT / "dist" / release / target
    raw = subprocess.check_output([args.go, "run", "./scripts/store_install", "--archive", str(out / f"codexgate_{release}_{goos}_{goarch}.zip"),
                                   "--checksums", str(out / "checksums.txt"), "--version", release, "--root", str(ROOT / ".runs")], cwd=ROOT, env=env, text=True)
    installed = json.loads(raw)
    write_new_json(out / "store-install.json", {k: v for k, v in installed.items() if k != "path"})
    shutil.copy2(build / "build-info.json", out / "build-info.json")
    if goos == "linux":
        shutil.copy2(build / "runtime-compatibility.json", out / "runtime-compatibility.json")
    subprocess.run([sys.executable, "scripts/smoke_cpa.py", "--core", str(core), "--library", installed["path"],
                    "--expected-version", release, "--report", str(out / "smoke.json")], cwd=ROOT, env=env, check=True)


if __name__ == "__main__":
    main()
